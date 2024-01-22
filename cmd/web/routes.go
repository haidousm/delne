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

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.health)

	router.HandlerFunc(http.MethodGet, "/v1/proxies", app.listProxies)
	router.HandlerFunc(http.MethodPost, "/v1/proxies/register", app.registerProxy)

	router.HandlerFunc(http.MethodGet, "/", app.proxyRequest)

	component := ProxyTable(*app.proxy)
	router.Handler(http.MethodGet, "/ui/proxies", templ.Handler(component))

	standardMiddleware := alice.New(app.recoverPanic, app.logRequest, app.secureHeaders)
	return standardMiddleware.Then(router)
}
