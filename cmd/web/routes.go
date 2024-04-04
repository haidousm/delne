package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()
	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/services", http.StatusSeeOther)
	})

	fileServer := http.FileServer(http.Dir("./ui/static/"))
	router.Handler(http.MethodGet, "/admin/static/*filepath", http.StripPrefix("/static", fileServer))

	router.HandlerFunc(http.MethodGet, "/admin/api/healthcheck", app.health)

	/**
	 * Services
	 */

	router.HandlerFunc(http.MethodPost, "/admin/api/services", app.createService)
	router.HandlerFunc(http.MethodDelete, "/admin/api/services/:name", app.deleteService)
	router.HandlerFunc(http.MethodPut, "/admin/api/services/:name", app.updateService)

	router.HandlerFunc(http.MethodPost, "/admin/api/services/:name/start", app.startService)
	router.HandlerFunc(http.MethodPost, "/admin/api/services/:name/stop", app.stopService)

	router.HandlerFunc(http.MethodGet, "/admin/services", app.servicesTableView)
	router.HandlerFunc(http.MethodGet, "/admin/service/new", app.createServiceFormView)
	router.HandlerFunc(http.MethodGet, "/admin/services/:name/edit", app.editServiceView)

	standardMiddleware := alice.New(app.recoverPanic, app.logRequest) // app.secureHeaders
	return standardMiddleware.Then(router)
}
