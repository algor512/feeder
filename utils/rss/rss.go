package rss

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/algor512/feeder/utils"
	"github.com/mmcdole/gofeed"
)

type RssSource struct {
	Name string
	Url string
	Tags []string
}

func (s *RssSource) Collect(c chan<- utils.Record, after time.Time) error {
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
		date := item.PublishedParsed.Local()
		if !date.Before(after) {
			rec := utils.Record{
				CollectDate: time.Now(),
				Date: item.PublishedParsed.In(time.Local),
				Title: item.Title,
				Url: item.Link,
				Tags: s.Tags,
				Source: s.Name,
			}
			c <- rec
		}
	}

	return nil
}
