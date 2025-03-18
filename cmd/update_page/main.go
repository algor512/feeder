package main

import (
	"bufio"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Record struct {
	Date time.Time
	Source string
	Title string
	Url string
}

type Data struct {
	StatusPresented bool
	StatusDate time.Time
	StatusSourcesNumConfigured int
	StatusSourcesNumErrors int
	StatusSourcesErrorList []string

	Records []Record
}

func main() {
	tplFile := flag.String("template", "page.html", "template file")
	records := flag.String("records", "records.tsv", "records file")
	status := flag.String("status", "", "status file")
	flag.Parse()

	if len(*records) == 0 {
		fmt.Fprintf(os.Stderr, "records file must be specified\n")
		flag.PrintDefaults()
		return
	}
	if len(*tplFile) == 0 {
		fmt.Fprintf(os.Stderr, "template file must be specified\n")
		flag.PrintDefaults()
		return
	}

	var data Data
	tpl, err := template.ParseFiles(*tplFile)
	if err != nil {
		log.Fatalf("Error while parsing template %s: %v\n", *tplFile, err)
		panic(err)
	}

	if len(*status) > 0 {
		file, err := os.Open(*status)
		if err != nil {
			log.Fatalf("Cannot open file %s\n", *status)
		}
		scanner := bufio.NewScanner(file)
		if !scanner.Scan() {
			log.Fatalln("Status file contains no lines")
		}
		data.StatusPresented = true

		parts := strings.Split(scanner.Text(), "\t")
		if len(parts) != 3 {
			log.Fatalln("Status file's first line must consist of 3 numbers")
		}
		var err1, err2, err3 error
		var nsrc, nerrs int
		var t int64
		t, err1 = strconv.ParseInt(parts[0], 10, 64)
		nerrs, err2 = strconv.Atoi(parts[1])
		nsrc, err3 = strconv.Atoi(parts[2])
		if err1 != nil || err2 != nil || err3 != nil {
			log.Fatalln("Status file's first line must consist of 3 numbers")
		}
		data.StatusDate = time.Unix(t, 0)
		data.StatusSourcesNumConfigured = nsrc
		data.StatusSourcesNumErrors = nerrs

		data.StatusSourcesErrorList = make([]string, 0)
		for scanner.Scan() {
			data.StatusSourcesErrorList = append(data.StatusSourcesErrorList, scanner.Text())
		}
	}

	data.Records = make([]Record, 0)
	file, err := os.Open(*records)
	if err != nil {
		log.Fatalf("Cannot open file %s\n", *records)
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), "\t")
		if len(parts) != 4 {
			log.Fatalln("Status file's first line must consist of 3 numbers")
		}

		record := Record{Source: parts[1], Url: parts[2], Title: parts[3]}
		t, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			log.Fatalf("'%s' is not a timestamp\n", parts[0])
		}
		record.Date = time.Unix(t, 0)
		data.Records = append(data.Records, record)
	}

	err = tpl.Execute(os.Stdout, data)
	if err != nil {
		log.Fatalf("Error while executing the template: %s\n", err)
	}
}
