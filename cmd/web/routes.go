package main

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()
	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.notFound(w)
	})

	fileServer := http.FileServer(http.Dir("./ui/static/"))
	router.Handler(http.MethodGet, "/static/*filepath", http.StripPrefix("/static", fileServer))

	router.HandlerFunc(http.MethodGet, "/", app.proxyRequest)

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.health)

	router.HandlerFunc(http.MethodGet, "/v1/proxies", app.listProxies)
	router.HandlerFunc(http.MethodPost, "/v1/proxies/register", app.registerProxy)

	component := ProxyPage(*app.proxy)
	router.Handler(http.MethodGet, "/ui/proxies", templ.Handler(component))
	router.HandlerFunc(http.MethodGet, "/ui/proxies/:host/edit", app.editProxyForm)

	standardMiddleware := alice.New(app.recoverPanic, app.logRequest) // app.secureHeaders
	return standardMiddleware.Then(router)
}
