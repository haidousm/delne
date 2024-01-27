package main

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/haidousm/delne/internal/docker"
)

func (app *application) servicesTableView(w http.ResponseWriter, r *http.Request) {
	services := app.proxy.Services
	component := ServicesDashboard(services)
	templ.Handler(component).ServeHTTP(w, r)
}

func (app *application) createServiceFormView(w http.ResponseWriter, r *http.Request) {
	onlyPartial := r.Header.Get("HX-Request") == "true"
	if !onlyPartial {
		http.Redirect(w, r, "/admin/services", http.StatusSeeOther)
		return
	}

	component := servicesTable(app.proxy.Services, true)
	component.Render(r.Context(), w)
}

func (app *application) createService(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	name := r.PostForm.Get("name")
	image := r.PostForm.Get("image")
	host := r.PostForm.Get("host")

	if name == "" || image == "" || host == "" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	imageObj := docker.Image{}
	imageObj.ParseString(image)

	service := docker.Service{
		Name:  name,
		Image: imageObj,
		Hosts: []string{host},
	}

	app.proxy.Services = append(app.proxy.Services, service)

	onlyPartial := r.Header.Get("HX-Request") == "true"
	if !onlyPartial {
		http.Redirect(w, r, "/admin/services", http.StatusSeeOther)
		return
	}

	component := servicesTableRow(service)
	component.Render(r.Context(), w)
}
