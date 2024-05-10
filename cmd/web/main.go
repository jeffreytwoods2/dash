package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"
	_ "github.com/lib/pq"
	"mc.jwoods.dev/internal/models"
)

type config struct {
	port int
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  time.Duration
	}
	subscriberMessageBuffer int
}

type application struct {
	logger         *slog.Logger
	templateCache  map[string]*template.Template
	subscribersMu  sync.Mutex
	subscribers    map[*subscriber]struct{}
	config         config
	sessionManager *scs.SessionManager
	users          *models.UserModel
	formDecoder    *form.Decoder
}

func main() {
	cfg := config{}

	flag.IntVar(&cfg.port, "port", 5001, "HTTP network address")

	flag.StringVar(&cfg.db.dsn, "db-dsn", "", "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.DurationVar(&cfg.db.maxIdleTime, "db-max-idle-time", 15*time.Minute, "PostgreSQL max connection idle time")

	flag.IntVar(&cfg.subscriberMessageBuffer, "buffer", 16, "Max number of queued messages for a subscriber")

	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	db, err := openDB(cfg)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	defer db.Close()

	logger.Info("database connection pool established")

	templateCache, err := newTemplateCache()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	formDecoder := form.NewDecoder()

	sessionManager := scs.New()
	sessionManager.Store = postgresstore.New(db)
	sessionManager.Lifetime = 24 * time.Hour

	app := &application{
		config:         cfg,
		logger:         logger,
		templateCache:  templateCache,
		subscribers:    make(map[*subscriber]struct{}),
		sessionManager: sessionManager,
		users:          &models.UserModel{DB: db},
		formDecoder:    formDecoder,
	}

	srv := &http.Server{
		Addr:     fmt.Sprintf(":%d", app.config.port),
		Handler:  app.routes(),
		ErrorLog: slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	go func() {
		for {
			if len(app.subscribers) > 0 {
				time.Sleep(2 * time.Second)

				app.renderOnline()
			}
		}
	}()

	logger.Info("starting web server", "port", app.config.port)

	err = srv.ListenAndServe()
	logger.Error(err.Error())
	os.Exit(1)
}

func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	db.SetMaxIdleConns(cfg.db.maxIdleConns)
	db.SetConnMaxIdleTime(cfg.db.maxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
