package main

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
)

const PROXY_ADMIN_PATH = "/admin"

type Proxy struct {
	Target   map[string]string
	RevProxy map[string]*httputil.ReverseProxy
}

func (app *application) proxyRoutes() http.Handler {

	// use stdlib mux instead of httprouter
	// because we want to proxy all requests regardless of path
	mux := http.NewServeMux()
	mux.HandleFunc("/", app.proxyRequest)
	standardMiddleware := alice.New(app.recoverPanic, app.logRequest)
	return standardMiddleware.Then(mux)
}

func (app *application) proxyRequest(w http.ResponseWriter, r *http.Request) {

	if len(r.URL.Path) >= len(PROXY_ADMIN_PATH) && r.URL.Path[:len(PROXY_ADMIN_PATH)] == PROXY_ADMIN_PATH {
		app.adminHandler.ServeHTTP(w, r)
		return
	}

	p := app.proxy

	host := r.Host
	// if we already have a rev proxy for this host setup
	if rev, ok := p.RevProxy[host]; ok {
		app.logger.Debug("proxying request to existing rev proxy")
		rev.ServeHTTP(w, r)
		return
	}

	// otherwise, create one
	if target, ok := p.Target[host]; ok {
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
	err := errors.New("forbidden host")
	app.serverError(w, r, err)
}

func (app *application) listProxies(w http.ResponseWriter, r *http.Request) {
	p := app.proxy
	for host, target := range p.Target {
		w.Write([]byte(host + " -> " + target + "\n"))
	}
}

func (app *application) registerProxy(w http.ResponseWriter, r *http.Request) {
	p := app.proxy

	if err := r.ParseForm(); err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	host := r.Form.Get("host")
	target := r.Form.Get("target")

	if host == "" || target == "" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	p.Target[host] = target

	onlyPartial := r.Header.Get("HX-Request") == "true"
	if !onlyPartial {
		http.Redirect(w, r, "/admin/proxies", http.StatusSeeOther)
		return
	}

	component := proxyTableRow(host, target)
	component.Render(r.Context(), w)
}

func (app *application) editProxy(w http.ResponseWriter, r *http.Request) {
	p := app.proxy
	params := httprouter.ParamsFromContext(r.Context())

	if err := r.ParseForm(); err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	currentHost := params.ByName("host")

	newHost := r.Form.Get("host")
	target := r.Form.Get("target")

	if newHost == "" || target == "" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	delete(p.Target, currentHost)
	p.Target[newHost] = target

	onlyPartial := r.Header.Get("HX-Request") == "true"
	if !onlyPartial {
		http.Redirect(w, r, "/admin/proxies", http.StatusSeeOther)
		return
	}

	component := proxyTableRow(newHost, target)
	component.Render(r.Context(), w)
}

func (app *application) editProxyForm(w http.ResponseWriter, r *http.Request) {

	params := httprouter.ParamsFromContext(r.Context())

	host := params.ByName("host")
	target := app.proxy.Target[host]

	onlyPartial := r.Header.Get("HX-Request") == "true"
	if !onlyPartial {
		http.Redirect(w, r, "/admin/proxies", http.StatusSeeOther)
		return
	}

	component := editProxyForm(host, target)
	component.Render(r.Context(), w)
}

func (app *application) createProxyForm(w http.ResponseWriter, r *http.Request) {
	onlyPartial := r.Header.Get("HX-Request") == "true"
	if !onlyPartial {
		http.Redirect(w, r, "/admin/proxies", http.StatusSeeOther)
		return
	}

	component := proxyTable(*app.proxy, true)
	component.Render(r.Context(), w)
}
