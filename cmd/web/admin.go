package main

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
)

func (app *application) adminRoutes() http.Handler {
	router := httprouter.New()
	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.notFound(w)
	})

	fileServer := http.FileServer(http.Dir("./ui/static/"))
	router.Handler(http.MethodGet, "/static/*filepath", http.StripPrefix("/static", fileServer))

	router.HandlerFunc(http.MethodGet, "/admin/api/healthcheck", app.health)

	router.HandlerFunc(http.MethodGet, "/admin/api/proxies", app.listProxies)
	router.HandlerFunc(http.MethodPut, "/admin/api/proxies/:host", app.editProxy)
	router.HandlerFunc(http.MethodPost, "/admin/api/proxies", app.registerProxy)

	component := ProxyPage(*app.proxy)
	router.Handler(http.MethodGet, "/admin/proxies", templ.Handler(component))
	router.HandlerFunc(http.MethodGet, "/admin/proxies/:host/edit", app.editProxyForm)
	router.HandlerFunc(http.MethodGet, "/admin/proxy/new", app.createProxyForm)

	standardMiddleware := alice.New(app.recoverPanic, app.logRequest) // app.secureHeaders
	return standardMiddleware.Then(router)
}
