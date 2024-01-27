package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"os"
	"time"

	"github.com/haidousm/delne/internal/docker"
	"github.com/haidousm/delne/internal/vcs"
	"github.com/justinas/alice"
)

type config struct {
	port  int
	env   string
	debug bool
}

type application struct {
	config  config
	logger  *slog.Logger
	proxy   *Proxy
	dClient *docker.Client
}

var (
	version = vcs.Version()
)

func main() {
	var cfg config
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
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
			Services: []*docker.Service{
				{
					Name:  "foo",
					Hosts: []string{"foo.com"},
					Port:  "8020",
					Image: docker.Image{
						Repository: "docker.io/haidousm",
						Name:       "foo",
						Tag:        "latest",
					},
				},
			},
		},
		dClient: dClient,
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
