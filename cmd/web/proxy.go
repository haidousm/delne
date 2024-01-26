package main

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Proxy struct {
	Target   map[string]string
	RevProxy map[string]*httputil.ReverseProxy
}

func (app *application) proxyRequest(w http.ResponseWriter, r *http.Request) {
	p := app.proxy

	host := r.Host
	if r.URL.Path != "/" {
		host += r.URL.Path
	}

	// prefix matching
	for k := range p.Target {
		if len(k) > len(host) {
			continue
		}
		if k == host[:len(k)] {
			r.URL.Path = host[len(k):]
			if rev, ok := p.RevProxy[k]; ok {
				app.logger.Debug("proxying request to existing rev proxy")
				rev.ServeHTTP(w, r)
				return
			}
			if target, ok := p.Target[k]; ok {
				app.logger.Debug("proxying request to new rev proxy")
				remote, err := url.Parse(target)
				if err != nil {
					app.serverError(w, r, err)
					return
				}

				rev := httputil.NewSingleHostReverseProxy(remote)
				p.RevProxy[host] = rev
				rev.ServeHTTP(w, r)
				return
			}
		}
	}

	err := errors.New("forbidden host")
	app.logger.Error(err.Error())
	app.notFound(w)
}
