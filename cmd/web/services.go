package main

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/haidousm/delne/internal/models"
	"github.com/julienschmidt/httprouter"
)

func (app *application) servicesTableView(w http.ResponseWriter, r *http.Request) {
	services, err := app.services.GetAll()
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	images, err := app.images.GetAll()
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	component := ServicesDashboard(services, images)
	templ.Handler(component).ServeHTTP(w, r)
}

func (app *application) createServiceFormView(w http.ResponseWriter, r *http.Request) {
	onlyPartial := r.Header.Get("HX-Request") == "true"
	if !onlyPartial {
		http.Redirect(w, r, "/admin/services", http.StatusSeeOther)
		return
	}

	services, err := app.services.GetAll()
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	images, err := app.images.GetAll()
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	component := servicesTable(services, images, true)
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

	imageObj := models.Image{}
	imageObj.ParseString(image)

	imageId, err := app.images.Insert(imageObj.Repository, imageObj.Name, imageObj.Tag)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	network := "delne" //temp, should be configurable
	service := models.Service{
		Name:    name,
		ImageID: &imageId,
		Hosts:   []string{host},
		Network: &network,
		Status:  models.PULLING,
	}

	app.logger.Debug("creating service", "name", name, "image", image, "host", host)
	serviceId, err := app.services.Insert(name, []string{host}, imageId, network)
	service.ID = serviceId

	if err != nil {
		app.serverError(w, r, err)
		return
	}

	go app.createContainerForService(&service, &imageObj)

	onlyPartial := r.Header.Get("HX-Request") == "true"
	if !onlyPartial {
		http.Redirect(w, r, "/admin/services", http.StatusSeeOther)
		return
	}

	component := servicesTableRow(service, imageObj)
	component.Render(r.Context(), w)
}

func (app *application) deleteService(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	name := params.ByName("name")

	if name == "" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	service, err := app.services.GetByName(name)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.dClient.RemoveContainer(*service)
	app.RemoveService(*service)

	err = app.services.Delete(service.ID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	onlyPartial := r.Header.Get("HX-Request") == "true"
	if !onlyPartial {
		http.Redirect(w, r, "/admin/services", http.StatusSeeOther)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (app *application) startService(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	name := params.ByName("name")

	if name == "" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	service, err := app.services.GetByName(name)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	if service.Status == models.RUNNING {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	err = app.dClient.StartContainer(*service)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	service.Status = models.RUNNING
	app.services.UpdateStatus(service.ID, models.RUNNING)

	onlyPartial := r.Header.Get("HX-Request") == "true"
	if !onlyPartial {
		http.Redirect(w, r, "/admin/services", http.StatusSeeOther)
		return
	}

	image, err := app.images.Get(*service.ImageID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	component := servicesTableRow(*service, *image)
	component.Render(r.Context(), w)
}

func (app *application) stopService(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	name := params.ByName("name")

	if name == "" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	service, err := app.services.GetByName(name)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	if service.Status != models.RUNNING {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	err = app.dClient.StopContainer(*service)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	service.Status = models.STOPPED
	app.services.UpdateStatus(service.ID, models.STOPPED)

	onlyPartial := r.Header.Get("HX-Request") == "true"
	if !onlyPartial {
		http.Redirect(w, r, "/admin/services", http.StatusSeeOther)
		return
	}

	image, err := app.images.Get(*service.ImageID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	component := servicesTableRow(*service, *image)
	component.Render(r.Context(), w)
}

func (app *application) createContainerForService(service *models.Service, image *models.Image) {

	resp, err := app.dClient.CreateContainer(*service, *image)

	if err != nil {
		app.logger.Error(err.Error())
		return
	}
	app.services.UpdateStatus(service.ID, models.CREATED)
	app.services.UpdateContainerId(service.ID, resp.ID)

	service, err = app.services.Get(service.ID)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	app.logger.Debug("created container", "id", resp.ID)

	err = app.dClient.StartContainer(*service)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}
	app.services.UpdateStatus(service.ID, models.RUNNING)
	app.logger.Debug("started container", "id", resp.ID)

	for _, host := range service.Hosts {
		app.proxy.Target[host] = service.Name
	}
}
