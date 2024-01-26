package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

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
