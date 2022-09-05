package telegram

import (
	"bytes"
	"container/list"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/algor512/feeder/utils"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type TelegramSource struct {
	Name string
	Channel string
	Tags []string
}

func (s *TelegramSource) Collect(ch chan<- utils.Record, after time.Time) error {
	url := "https://t.me/s/" + s.Channel
	done := false
	for !done {
		res, err := http.Get(url)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		root, err := html.Parse(res.Body)
		if err != nil {
			return err
		}

		recordChan := make(chan utils.Record)
		errorChan := make(chan error)

		go parsePage(root, recordChan, errorChan)
		for recordChan != nil {
			select {
			case rec, ok := <-recordChan:
				if !ok {
					recordChan = nil
					break
				}

				if rec.Date.Before(after) {
					done = true
				} else {
					if len(rec.Title) > 0 && len(rec.Url) > 0 {
						rec.Tags = s.Tags
						rec.Source = s.Name
						ch <- rec
					} else {
						log.Printf("wrong record %v", rec)
					}
				}
			case err := <-errorChan:
				if err != nil {
					return err
				}
			}
		}

		linkPrev := findTag(root, func(t *html.Node) bool { return t.Data == "link" && getAttr(t, "rel") == "prev" })
		if linkPrev == nil {
			break
		}
		url = "https://t.me" + getAttr(linkPrev, "href")
	}

	return nil
}

func parsePage(root *html.Node, out chan<- utils.Record, errs chan<- error) {
	defer close(out)
	defer close(errs)

	container := findTag(root, func(t *html.Node) bool { return t.Data == "section" && hasClass(t, "tgme_channel_history") })
	if container == nil {
		errs <- fmt.Errorf("cannot find telegram messages container")
		return
	}

	for block := container.FirstChild; block != nil; block = block.NextSibling {
		if block.Type != html.ElementNode || !hasClass(block, "tgme_widget_message_wrap") {
			continue
		}
		rec, err := parseRecord(block)
		if err != nil {
			errs <- err
		} else {
			out <- rec
		}
	}
}

func parseRecord(block *html.Node) (utils.Record, error) {
	record := utils.Record{CollectDate: time.Now()}

	queue := list.New()
	queue.PushBack(block)

	for queue.Len() > 0 {
		e := queue.Front()
		tag := e.Value.(*html.Node)
		if tag.Type == html.ElementNode {
			switch {
			case tag.Data == "div" && hasClass(tag, "tgme_widget_message_text"):
				titleRunes := []rune(extractHeader(tag))
				if len(titleRunes) > 100 {
					titleRunes = append(titleRunes[:100], '.', '.', '.')
				}
				record.Title = string(titleRunes)
			case tag.Data == "a" && hasClass(tag, "tgme_widget_message_date"):
				record.Url = getAttr(tag, "href")
			case tag.Data == "time" && hasClass(tag, "time"):
				var err error
				record.Date, err = time.Parse(time.RFC3339, getAttr(tag, "datetime"))
				if err != nil {
					return record, err
				}
				record.Date = record.Date.Local()
			}
		}
		for c := tag.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				queue.PushBack(c)
			}
		}
		queue.Remove(e)
	}

	return record, nil
}

func extractHeader(root *html.Node) string {
	buf := &bytes.Buffer{}
	stack := list.New()
	stack.PushBack(root)

Processing:
	for stack.Len() > 0 {
		e := stack.Back()
		stack.Remove(e)
		node := e.Value.(*html.Node)

		switch {
		case node.Type == html.TextNode:
			buf.WriteString(node.Data)
		case node.Type == html.ElementNode:
			if node.DataAtom == atom.Br {
				break Processing
			}
			for c := node.LastChild; c != nil; c = c.PrevSibling {
				stack.PushBack(c)
			}
		}
	}
	return buf.String()
}

func getAttr(node *html.Node, attrName string) string {
	for _, attr := range node.Attr {
		if attr.Key == attrName {
			return attr.Val
		}
	}
	return ""
}

func hasClass(node *html.Node, cname string) bool {
	for _, attr := range node.Attr {
		if attr.Key == "class" && strings.Contains(attr.Val, cname) {
			return true
		}
	}
	return false
}

func findTag(root *html.Node, check func(node *html.Node) bool) *html.Node {
	queue := list.New()
	queue.PushBack(root)

	for queue.Len() > 0 {
		e := queue.Front()
		tag := e.Value.(*html.Node)
		if tag.Type == html.ElementNode {
			if check(tag) {
				return tag
			}
		}
		for c := tag.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				queue.PushBack(c)
			}
		}
		queue.Remove(e)
	}
	return nil
}
