package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mmcdole/gofeed"
)

type Source struct {
	Name string
	Url string
}

type Item struct {
	Date time.Time
	Title string
	Url string
	Source string
}

func (s *Source) collect(c chan<- Item) error {
	fp := gofeed.NewParser()
	fp.Client = &http.Client{
		Timeout: 10*time.Second,
	}
	if strings.Contains(s.Url, "reddit.com") {
		fp.UserAgent = "RSS Reader 1.0 by /u/algor512"
		fp.Client.Transport = &http.Transport{TLSClientConfig: &tls.Config{}}
	}
	feed, err := fp.ParseURL(s.Url)
	if err != nil {
		return fmt.Errorf("%s: %s", s.Url, err)
	}
	for _, item := range feed.Items {
		rec := Item{
			Date:   *item.PublishedParsed,
			Title:  item.Title,
			Url:    item.Link,
			Source: s.Name,
		}
		c <- rec
	}
	return nil
}

func (item *Item) format() string {
	ws := regexp.MustCompile(`\s+`)
	return fmt.Sprintf("%d\t%s\t%s\t%s", item.Date.Unix(), item.Source, item.Url, ws.ReplaceAllString(item.Title, " "))
}

func parseConfigFile(config string) ([]Source, error) {
	file, err := os.Open(config)
    if err != nil {
        return nil, err
	}
    defer file.Close()

	sources := make([]Source, 0, 100)
	scanner := bufio.NewScanner(file)
	record := make(map[string]string)
	for lineno := 1; ; lineno++ {
		not_eof := scanner.Scan()
		line := strings.TrimSpace(scanner.Text())
		if !not_eof || (len(line) == 0) {
			var name, url string
			var ok bool
			if name, ok = record["name"]; !ok {
				return sources, fmt.Errorf("an error in line %s:%d: missing field 'name'", config, lineno)
			}
			if url, ok = record["url"]; !ok {
				return sources, fmt.Errorf("an error in line %s:%d: missing field 'url'", config, lineno)
			}
			sources = append(sources, Source{Name: name, Url: url})
		} else {
			field, value, found := strings.Cut(line, ": ")
			if !found {
				return sources, fmt.Errorf("an error in line %s:%d: unknown format", config, lineno)
			}
			record[field] = value
		}

		if !not_eof {
			break
		}
	}

	return sources, nil
}


func main() {
	config := flag.String("config", "config.rec", "config file")
	status := flag.String("status", "", "status file")
	flag.Parse()
	if len(*config) == 0 {
		fmt.Fprintf(os.Stderr, "config file must be specified\n")
		flag.PrintDefaults()
		return
	}

	sources, err := parseConfigFile(*config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while parsing config file")
		panic(err)
	}

	ch := make(chan Item)
	errs := make(chan string, len(sources))
	var wg sync.WaitGroup
	wg.Add(len(sources))
	for _, src := range sources {
		go func(src Source) {
			defer wg.Done()
			err := src.collect(ch)
			if err != nil {
				errs <- src.Name
				log.Printf("error while processing %+v: %s", src, err)
			}
		}(src)
	}
	go func() {
		wg.Wait()
		close(ch)
		close(errs)
	}()

	for item := range ch {
		fmt.Println(item.format())
	}

	if len(*status) > 0 {
		file, err := os.Create(*status)
		if err != nil {
			panic(err)
		}
		defer file.Close()

		fmt.Fprintf(file, "%d\t%d\t%d\n", time.Now().Unix(), len(errs), len(sources))
		for e := range errs {
			fmt.Fprintln(file, e)
		}
	}
}
