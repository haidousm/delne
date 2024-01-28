package main

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/haidousm/delne/internal/docker"
)

type Proxy struct {
	Target   map[string]string
	RevProxy map[string]*httputil.ReverseProxy
	Services []*docker.Service
}

func (app *application) GetService(name string) *docker.Service {
	proxy := app.proxy
	for _, s := range proxy.Services {
		if s.Name == name {
			return s
		}
	}
	return nil
}

func (app *application) RemoveService(name string) error {
	service := app.GetService(name)
	if service == nil {
		return errors.New("service not found")
	}

	proxy := app.proxy
	for i, s := range proxy.Services {
		if s.Name == name {
			proxy.Services = append(proxy.Services[:i], proxy.Services[i+1:]...)
			break
		}
	}

	for k, v := range proxy.Target {
		if v == name {
			delete(proxy.Target, k)
			break
		}
	}

	for k := range proxy.RevProxy {
		if k == name {
			delete(proxy.RevProxy, k)
			break
		}
	}

	err := app.dClient.RemoveContainer(*service)
	return err
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

				// find service with name == target
				var service *docker.Service
				for _, s := range p.Services {
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

	err := errors.New("forbidden host")
	app.logger.Error(err.Error())
	app.notFound(w)
}
