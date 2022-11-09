package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/algor512/feeder/utils"
	"github.com/algor512/feeder/utils/rss"
	// "github.com/algor512/feeder/utils/telegram"
)

func createRssSource(name string, record map[string]string) (*rss.RssSource, error) {
	url, ok_url := record["url"]
	tags, ok_tags := record["tags"]
	if !ok_url {
		return nil, fmt.Errorf("an error in record '%s': missing field 'url'", name)
	}
	if !ok_tags {
		return nil, fmt.Errorf("an error in record '%s': missing field 'tags'", name)
	}
	return &rss.RssSource{Name: name, Url: url, Tags: append(strings.Split(tags, ", "), "rss")}, nil
}

func parseConfigFile(config string) ([]utils.Source, error) {
	file, err := os.Open(config)
    if err != nil {
        return nil, err
	}
    defer file.Close()

	sources := make([]utils.Source, 0, 100)
	scanner := bufio.NewScanner(file)
	lineno := 1
	record := make(map[string]string)
	for {
		not_eof := scanner.Scan()
		line := strings.TrimSpace(scanner.Text())
		if !not_eof || (len(line) == 0) {
			name, ok_name := record["name"]
			if !ok_name {
				return sources, fmt.Errorf("an error in line %s:%d: missing field 'name'", config, lineno)
			}
			if typ, ok := record["type"]; ok {
				var src utils.Source
				var err error
				switch typ {
				case "rss":
					src, err = createRssSource(name, record)
				default:
					return sources, fmt.Errorf("an error in line %s:%d: unknown type '%s'", config, lineno, typ)
				}
				if err != nil {
					return sources, err
				}
				sources = append(sources, src)
			} else {
				return sources, fmt.Errorf("an error in line %s:%d: field 'type' not found", config, lineno)
			}
		} else {
			field, value, found := strings.Cut(line, ": ")
			if !found {
				return sources, fmt.Errorf("an error in line %s:%d: unknown format", config, lineno)
			}
			record[field] = value
		}

		lineno += 1
		if !not_eof {
			break
		}
	}

	return sources, nil
}
