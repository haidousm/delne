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

	network := "delne" //temp, should be configurable
	service := docker.Service{
		Name:    name,
		Image:   imageObj,
		Hosts:   []string{host},
		Network: network,
	}

	app.logger.Debug("creating service", "name", name, "image", image, "host", host)
	app.proxy.Services = append(app.proxy.Services, &service)

	for _, host := range service.Hosts {
		app.proxy.Target[host] = service.Name
	}

	onlyPartial := r.Header.Get("HX-Request") == "true"
	if !onlyPartial {
		http.Redirect(w, r, "/admin/services", http.StatusSeeOther)
		return
	}

	go func() {
		resp, err := app.dClient.CreateContainer(service)

		if err != nil {
			app.logger.Error(err.Error())
			return
		}

		service.ContainerId = resp.ID
		app.logger.Debug("created container", "id", resp.ID)

		err = app.dClient.StartContainer(service)
		if err != nil {
			app.logger.Error(err.Error())
			return
		}
		app.logger.Debug("started container", "id", resp.ID)
	}()

	component := servicesTableRow(service)
	component.Render(r.Context(), w)
}
