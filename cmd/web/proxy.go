package main

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/julienschmidt/httprouter"
)

type Proxy struct {
	Target   map[string]string
	RevProxy map[string]*httputil.ReverseProxy
}

func (app *application) proxyRequest(w http.ResponseWriter, r *http.Request) {
	p := app.proxy

	host := r.Host
	// if we already have a rev proxy for this host setup
	if rev, ok := p.RevProxy[host]; ok {
		rev.ServeHTTP(w, r)
		return
	}

	// otherwise, create one
	if target, ok := p.Target[host]; ok {
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

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) editProxyForm(w http.ResponseWriter, r *http.Request) {

	params := httprouter.ParamsFromContext(r.Context())

	host := params.ByName("host")
	target := app.proxy.Target[host]

	component := editProxyForm(host, target)
	component.Render(r.Context(), w)
}
