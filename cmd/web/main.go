package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/foomo/simplecert"
	"github.com/foomo/tlsconfig"
	"github.com/haidousm/delne/internal/docker"
	"github.com/haidousm/delne/internal/models"
	"github.com/haidousm/delne/internal/vcs"
	"github.com/justinas/alice"
	_ "github.com/mattn/go-sqlite3"
)

type SSLConfig struct {
	Local    bool
	Email    string
	CacheDir string

	Domains []string
}

type config struct {
	Env   string
	Debug bool
	DSN   string
	SSL   SSLConfig
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
var (
	cfgFile = "delne.toml"
)

func main() {
	displayVersion := flag.Bool("version", false, "Display version and exit")
	flag.Parse()

	if *displayVersion {
		fmt.Printf("Version:\t%s\n", version)
		os.Exit(0)
	}

	var cfg config
	_, err := toml.DecodeFile(cfgFile, &cfg)
	if err != nil {
		fmt.Printf(err.Error())
		os.Exit(1)
	}

	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))

	db, err := openDB(cfg.DSN)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer db.Close()

	dClient, err := docker.NewClient(logger)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	app := &application{
		config: cfg,
		logger: logger,
		proxy: &Proxy{
			Target: map[string]string{
				"foo.local/test": "http://localhost:8020",
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
		Addr:         fmt.Sprintf(":%d", 443),
		Handler:      standardMiddleware.Then(mux),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	// app.rebuildProxyFromDB()

	listenAndServeTLS(srv, app)
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

func listenAndServeTLS(srv *http.Server, app *application) {

	scCfg := simplecert.Default
	scCfg.Local = app.config.SSL.Local

	scCfg.SSLEmail = app.config.SSL.Email
	scCfg.CacheDir = app.config.SSL.CacheDir

	scCfg.Domains = app.config.SSL.Domains

	certLoader, err := simplecert.Init(scCfg, nil)
	if err != nil {
		log.Fatal("simplecert init failed: ", err)
	}

	app.logger.Debug("starting redir from :80 to :443", "env", app.config.Env)
	errChan := make(chan error)
	go func() {
		errChan <- http.ListenAndServe(":80", http.HandlerFunc(simplecert.Redirect))
	}()

	tlsconf := tlsconfig.NewServerTLSConfig(tlsconfig.TLSModeServerStrict)
	tlsconf.GetCertificate = certLoader.GetCertificateFunc()
	srv.TLSConfig = tlsconf
	app.logger.Debug("starting server at :443", "env", app.config.Env)
	go func() {
		errChan <- srv.ListenAndServeTLS("", "")
	}()
	log.Fatal(<-errChan)
}
