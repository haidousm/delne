package main

import (
	"errors"

	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/haidousm/delne/internal/models"
)

type Proxy struct {
	Target   map[string]string
	RevProxy map[string]*httputil.ReverseProxy
}

func (p *Proxy) GetDomains() []string {
	uniqueDomains := make(map[string]bool)

	for targetKey := range p.Target {
		domain := parseDomain(targetKey)
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

func parseDomain(input string) string {
	input = strings.TrimSpace(input)

	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		parsedURL, err := url.Parse(input)
		if err == nil {
			return parsedURL.Hostname()
		}
	}

	parts := strings.Split(input, "/")
	domain := parts[0]

	if colonIndex := strings.Index(domain, ":"); colonIndex != -1 {
		domain = domain[:colonIndex]
	}
	return domain
}

func (app *application) AddTargetsFromService(service models.Service, isStartup bool) {
	for _, host := range service.Hosts {
		app.proxy.Target[host] = service.Name
	}

	if isStartup {
		app.logger.Debug("skipping server reload because this is startup buddy")
	} else {
		app.reloadServerBecauseOfCertChange()
	}
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
		go app.createContainerForService(service, image, true)
	}
	for _, domain := range app.proxy.GetDomains() {
		app.config.SSL.Domains = append(app.config.SSL.Domains, domain)
	}
}
