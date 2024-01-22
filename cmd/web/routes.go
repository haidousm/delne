package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()
	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.notFound(w)
	})

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.health)
	router.HandlerFunc(http.MethodPost, "/v1/proxies/register", app.registerProxy)

	router.HandlerFunc(http.MethodGet, "/", app.proxyRequest)

	standardMiddleware := alice.New(app.recoverPanic, app.logRequest, app.secureHeaders)
	return standardMiddleware.Then(router)
}
