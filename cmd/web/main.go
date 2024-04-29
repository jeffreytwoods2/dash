package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"text/template"
)

type subscriber struct {
	msgs      chan []byte
	closeSlow func()
}

type application struct {
	logger                  *slog.Logger
	templateCache           map[string]*template.Template
	subscriberMessageBuffer int
	subscribersMu           sync.Mutex
	subscribers             map[*subscriber]struct{}
}

func main() {
	addr := flag.String("addr", ":5001", "HTTP network address")
	buffer := flag.Int("buffer", 16, "Max number of queued messages for a subscriber")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	templateCache, err := newTemplateCache()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	app := &application{
		logger:                  logger,
		templateCache:           templateCache,
		subscriberMessageBuffer: *buffer,
		subscribers:             make(map[*subscriber]struct{}),
	}

	srv := &http.Server{
		Addr:     *addr,
		Handler:  app.routes(),
		ErrorLog: slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	logger.Info("starting web server", "port", *addr)

	err = srv.ListenAndServe()
	logger.Error(err.Error())
	os.Exit(1)
}
