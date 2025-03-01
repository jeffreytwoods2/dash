package main

import (
	"bytes"
	"context"
	"database/sql"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"
	_ "github.com/lib/pq"
	"mc.jwoods.dev/internal/models"
	"mc.jwoods.dev/internal/vcs"
)

var version = vcs.Version()

type config struct {
	env  string
	port int
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  time.Duration
	}
	jwt           string
	rconPwd       string
	rconAddr      string
	serviceWorker struct {
		staticDir      string
		staticFileList []string
	}
	subscriberMessageBuffer int
	wca                     struct {
		user string
		pwd  string
	}
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

	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	flag.IntVar(&cfg.port, "port", 5001, "HTTP network address")
	flag.StringVar(&cfg.serviceWorker.staticDir, "static-dir", os.Getenv("STATIC_DIR"), "Path to the ui/static directory")
	flag.StringVar(&cfg.db.dsn, "db-dsn", "", "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.DurationVar(&cfg.db.maxIdleTime, "db-max-idle-time", 15*time.Minute, "PostgreSQL max connection idle time")
	flag.StringVar(&cfg.rconPwd, "rcon-pwd", os.Getenv("RCON_PWD"), "Password for server RCON")
	flag.StringVar(&cfg.rconAddr, "rcon-addr", os.Getenv("RCON_ADDR"), "Server address for RCON requests")
	flag.StringVar(&cfg.wca.user, "wca-user", os.Getenv("WCA_USER"), "WebCraftAPI user")
	flag.StringVar(&cfg.wca.pwd, "wca-pwd", os.Getenv("WCA_PWD"), "Password for WebCraftAPI user")
	flag.IntVar(&cfg.subscriberMessageBuffer, "buffer", 16, "Max number of queued messages for a subscriber")
	displayVersion := flag.Bool("version", false, "Display version and exit")
	flag.Parse()

	if *displayVersion {
		fmt.Printf("Version:\t%s\n", version)
		os.Exit(0)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err := cfg.buildStaticFileList()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

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
	sessionManager.Lifetime = 24 * 365 * time.Hour

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
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}

	err = app.getJWT()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	go func() {
		for {
			if len(app.subscribers) > 0 {
				time.Sleep(1 * time.Second)

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

// Get authentication JWT for the WebCraftAPI service
func (app *application) getJWT() error {
	credentials := fmt.Sprintf("{\"username\": \"%s\", \"password\": \"%s\"}", app.config.wca.user, app.config.wca.pwd)

	req, err := http.NewRequest("POST", "http://localhost:8000/api/authenticate", bytes.NewBuffer([]byte(credentials)))
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	app.config.jwt = strings.Trim(string(body), "\"")

	return nil
}
