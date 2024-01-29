package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"os"
	"time"

	"github.com/haidousm/delne/internal/docker"
	"github.com/haidousm/delne/internal/models"
	"github.com/haidousm/delne/internal/vcs"
	"github.com/justinas/alice"
	_ "github.com/mattn/go-sqlite3"
)

type config struct {
	port  int
	env   string
	debug bool
	dsn   string
}

type application struct {
	config  config
	logger  *slog.Logger
	proxy   *Proxy
	dClient *docker.Client

	images   models.ImageModelInterface
	services models.ServiceModelInterface
}

var (
	version = vcs.Version()
)

func main() {
	var cfg config

	dsn := flag.String(cfg.dsn, "file:delne.db", "SQLite3 data source name")
	flag.IntVar(&cfg.port, "port", 4000, "server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")

	displayVersion := flag.Bool("version", false, "Display version and exit")

	flag.Parse()

	if *displayVersion {
		fmt.Printf("Version:\t%s\n", version)
		os.Exit(0)
	}

	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))

	db, err := openDB(*dsn)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer db.Close()

	dClient, err := docker.NewClient()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	app := &application{
		config: cfg,
		logger: logger,
		proxy: &Proxy{
			Target: map[string]string{
				"foo.com/test": "http://localhost:8020",
			},
			RevProxy: make(map[string]*httputil.ReverseProxy),
		},
		images:   &models.ImageModel{DB: db},
		services: &models.ServiceModel{DB: db},
		dClient:  dClient,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/admin/", app.routes().ServeHTTP)
	mux.HandleFunc("/", app.proxyRequest)

	standardMiddleware := alice.New(app.recoverPanic, app.logRequest)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      standardMiddleware.Then(mux),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	logger.Info("starting server", "addr", srv.Addr, "env", cfg.env)

	err = srv.ListenAndServe()
	logger.Error(err.Error())
	os.Exit(1)
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
