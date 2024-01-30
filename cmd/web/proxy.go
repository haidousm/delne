package main

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/haidousm/delne/internal/models"
)

type Proxy struct {
	Target   map[string]string
	RevProxy map[string]*httputil.ReverseProxy
}

func (app *application) RemoveService(service models.Service) {
	for k, v := range app.proxy.Target {
		if v == service.Name {
			delete(app.proxy.Target, k)
			break
		}
	}

	for k := range app.proxy.RevProxy {
		if k == service.Name {
			delete(app.proxy.RevProxy, k)
			break
		}
	}
}

func (app *application) proxyRequest(w http.ResponseWriter, r *http.Request) {
	p := app.proxy

	host := r.Host
	if r.URL.Path != "/" {
		host += r.URL.Path
	}

	services, err := app.services.GetAll()
	if err != nil {
		app.serverError(w, r, err)
		return
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

				// find service with name == target
				var service *models.Service
				for _, s := range services {
					if s.Name == target {
						service = s
						break
					}
				}

				if service == nil {
					app.logger.Error("service not found", "name", target)
					app.notFound(w)
					return
				}

				sUrl := service.Url()
				remote, err := url.Parse(sUrl)
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

	err = errors.New("forbidden host")
	app.logger.Error(err.Error())
	app.notFound(w)
}
