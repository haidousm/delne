package main

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/docker/docker/api/types"
	"github.com/haidousm/delne/internal/models"
)

type Proxy struct {
	Target   map[string]string
	RevProxy map[string]*httputil.ReverseProxy
}

func (p *Proxy) GetDomains() []string {
	uniqueDomains := make(map[string]bool)

	for targetURL := range p.Target {
		parsedURL, err := url.Parse(targetURL)
		if err != nil {
			continue
		}

		domain := parsedURL.Hostname()
		if domain != "" {
			uniqueDomains[domain] = true
		}
	}

	result := make([]string, 0, len(uniqueDomains))
	for domain := range uniqueDomains {
		result = append(result, domain)
	}
	return result
}

func (app *application) AddTargetsFromService(service models.Service) {
	for _, host := range service.Hosts {
		app.proxy.Target[host] = service.Name
	}
	app.config.SSL.Domains = app.proxy.GetDomains()
	app.dcl.ReloadCerts(app.config.SSL)
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

func (app *application) rebuildProxyFromDB() {

	containers, err := app.dClient.ListContainers()
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	filtered := []types.Container{}
	for _, c := range containers {
		if c.HostConfig.NetworkMode == "delne" && c.Names[0][1:] != "delne" {
			filtered = append(filtered, c)
		}
	}

	for _, c := range filtered {
		app.dClient.RemoveContainerById(c.ID)
	}

	services, err := app.services.GetAll()
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	images, err := app.images.GetAll()
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	for _, service := range services {
		var image *models.Image
		for _, i := range images {
			if i.ID == *service.ImageID {
				image = i
				break
			}
		}
		go app.createContainerForService(service, image)
	}
}
