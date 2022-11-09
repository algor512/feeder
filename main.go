package main

import (
	"context"
	"embed"
	_ "embed"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/algor512/feeder/utils"
)

var (
	//go:embed embed/*
	files embed.FS
	pageTpl *template.Template
)

func writeFeed(w io.Writer, provider *utils.FeedProvider, pageName string) error {
	err := provider.WritePage(w, pageName)
	if err != nil {
		log.Println(err)
	}
	return err
}

func RequestLogger(hdl http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			method := r.Method
			uri := r.RequestURI
			t := time.Now()

			log.Printf("--> %s\t%s %s", ip, method, uri)
			ww := NewResponseWriterWrapper(w)
			hdl.ServeHTTP(ww, r)
			log.Printf("<-- %s %d %d ms", ip, ww.Status, time.Since(t).Milliseconds())
		},
	)
}

func main() {
	listen := flag.String("listen", "", "listen address")
	database := flag.String("db", "records.json", "database file")
	config := flag.String("cfg", "config.rec", "config file")
	location := flag.String("loc", "Europe/Moscow", "location")
	useHTTPS := flag.Bool("https", false, "use https")
	cer := flag.String("cer", "", "path to cer file")
	key := flag.String("key", "", "path to key file")

	freq := flag.Int("freq", 30, "update frequency (in minutes)")
	period := flag.Int("period", 14, "data storage period (in days)")
	flag.Parse();

	if len(*config) == 0 {
		fmt.Fprintf(os.Stderr, "config file must be specified\n")
		flag.PrintDefaults()
		return
	}
	if len(*listen) == 0 {
		fmt.Fprintf(os.Stderr, "listen address must be specified\n")
		flag.PrintDefaults()
		return
	}
	if *useHTTPS && (len(*cer) == 0 || len(*key) == 0) {
		fmt.Fprintf(os.Stderr, "cer and key must be specified when using https\n")
		flag.PrintDefaults()
		return
	}

	loc, err := time.LoadLocation(*location)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unknown location: %s\n", *location)
		return
	}
	time.Local = loc

	pageTpl, err = template.ParseFS(files, "embed/page.html")
	if err != nil {
		panic(err)
	}

	sources, err := parseConfigFile(*config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while parsing config file")
		panic(err)
	}

	provider, err := utils.NewFeedProvider(utils.FeedProviderCfg{
		DatabaseFile: *database,
		PageTemplate: pageTpl,
		Frequency: time.Duration(*freq) * time.Minute,
		Period: time.Duration(24 * *period) * time.Hour,
		Sources: sources,
		Logger: log.Default(),
	})
	if err != nil {
		panic(err)
	}

	log.Println("start server")
	mux := http.NewServeMux()
	mux.HandleFunc("/", func (w http.ResponseWriter, r *http.Request) {
		dir := path.Dir(r.URL.Path)
		if dir != "/" {
			http.Error(w, "wrong path", http.StatusInternalServerError)
			return
		}

		pageName := path.Base(r.URL.Path)
		if pageName == "/" {
			pageName = "rss"
		}
		err := writeFeed(w, provider, pageName)
		if err != nil {
			http.Error(w, "page not found", http.StatusNotFound)
			return
		}
	})

	server := &http.Server{Addr: *listen, Handler: RequestLogger(mux)}
	var wgServer sync.WaitGroup
	wgServer.Add(1)
	go func() {
		defer wgServer.Done()
		if *useHTTPS {
			err := server.ListenAndServeTLS(*cer, *key)
			if err != nil && err != http.ErrServerClosed {
				panic(err)
			}
		} else {
			err := server.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				panic(err)
			}
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-sigChan

	if err := server.Shutdown(context.Background()); err != nil {
		log.Printf("stop server error: %s", err)
		panic(err)
	}
	wgServer.Wait()
	log.Println("server stopped")

	provider.Stop()
}
