package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"sort"
	"sync"
	"time"
)

type Record struct {
	CollectDate time.Time
	Date time.Time
	Title string
	Url string
	Tags []string
	Source string
}

type FeedProviderCfg struct {
	DatabaseFile string
	PageTemplate *template.Template
	Frequency time.Duration
	Period time.Duration
	Sources []Source
	Logger *log.Logger
}

type FeedProvider struct {
	config FeedProviderCfg

	pages map[string]*bytes.Buffer
	records map[string]Record

	mutex sync.RWMutex
	ticker *time.Ticker
	tickerDone chan bool
	wgUpdates sync.WaitGroup
}

type TemplateCfg struct {
	Records []Record
	CurrentPage string
}

func NewFeedProvider(cfg FeedProviderCfg) (*FeedProvider, error) {
	obj := &FeedProvider{
	    config: cfg,
		pages: nil,
		records: nil,
	}
	if err := obj.loadRecords(); err != nil {
		return obj, err
	}
	if err := obj.generatePages(); err != nil {
		return obj, err
	}

	obj.ticker = time.NewTicker(obj.config.Frequency)
	obj.tickerDone = make(chan bool)
	obj.wgUpdates.Add(1)
	go func() {
		defer obj.wgUpdates.Done()
		obj.update()
		for {
			select {
			case <-obj.tickerDone:
				return
			case <-obj.ticker.C:
				obj.update()
			}
		}
	}()

	return obj, nil
}

func (obj *FeedProvider) WritePage(w io.Writer, pageName string) error {
	obj.mutex.RLock()
	defer obj.mutex.RUnlock()

	pageBuf, ok := obj.pages[pageName]
	if !ok {
		return fmt.Errorf("page %s not found", pageName)
	}
	_, err := w.Write(pageBuf.Bytes())
	return err
}

func (obj *FeedProvider) Stop() error {
	obj.config.Logger.Println("stopping...")
	obj.ticker.Stop()
	obj.tickerDone <- true
	close(obj.tickerDone)
	obj.wgUpdates.Wait()
	return nil
}

func (obj *FeedProvider) update() {
	obj.config.Logger.Println("starting database update")
	dateFrom := time.Now().Add(-obj.config.Period)
	ch := make(chan Record)

	defer func() {
		if r := recover(); r != nil {
			obj.config.Logger.Printf("recovered from error: %s", r)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(len(obj.config.Sources))
	for _, src := range obj.config.Sources {
		go func(src Source) {
			defer wg.Done()
			err := src.Collect(ch, dateFrom)
			if err != nil {
				obj.config.Logger.Printf("error while processing %+v: %s", src, err)
			}
		}(src)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()

	inserted, deleted := 0, 0

	for r := range ch {
		if _, ok := obj.records[r.Url]; !ok {
			obj.records[r.Url] = r
			inserted++
		}
	}

	for k, r := range obj.records {
		if r.Date.Before(dateFrom) {
			delete(obj.records, k)
			deleted++
		}
	}

	obj.config.Logger.Printf("update finished; inserted=%d, deleted=%d", inserted, deleted)

	obj.flushRecords()
	obj.generatePages()
}

func (obj *FeedProvider) loadRecords() error {
	if _, err := os.Stat(obj.config.DatabaseFile); errors.Is(err, os.ErrNotExist) {
		obj.records = make(map[string]Record)
		return nil
	}

	fin, err := os.Open(obj.config.DatabaseFile)
	if err != nil {
		return err
	}
	defer fin.Close()

	decoder := json.NewDecoder(fin)
	err = decoder.Decode(&obj.records)
	if err != nil {
		return err
	}
	obj.config.Logger.Printf("loaded %d records from file", len(obj.records))

	return nil
}

func (obj *FeedProvider) flushRecords() error {
	fout, err := os.Create(obj.config.DatabaseFile)
	if err != nil {
		return err
	}
	defer fout.Close()

	encoder := json.NewEncoder(fout)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(obj.records)
	if err != nil {
		return err
	}
	obj.config.Logger.Printf("saved %d records to file", len(obj.records))
	return nil
}

func (obj *FeedProvider) generatePages() error {
	recordPerTag := make(map[string][]Record)
	for _, rec := range obj.records {
		for _, tag := range rec.Tags {
			if recs, ok := recordPerTag[tag]; ok {
				recs = append(recs, rec)
				recordPerTag[tag] = recs
			} else {
				recs = []Record{rec,}
				recordPerTag[tag] = recs
			}
		}
	}
	obj.mutex.Lock()
	defer obj.mutex.Unlock()
	obj.pages = make(map[string]*bytes.Buffer)
	for tag, records := range recordPerTag {
		sort.Slice(records, func(i, j int) bool {
			return records[i].Date.After(records[j].Date)
		})
		obj.pages[tag] = new(bytes.Buffer)
		err := obj.config.PageTemplate.Execute(obj.pages[tag], TemplateCfg{Records: records, CurrentPage: tag})
		if err != nil {
			return err
		}
    }
	obj.config.Logger.Printf("generated %d pages", len(obj.pages))
	return nil
}
