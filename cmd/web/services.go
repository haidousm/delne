package main

import (
	"fmt"
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

func (app *application) editServiceView(w http.ResponseWriter, r *http.Request) {
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

	image, err := app.images.Get(*service.ImageID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	onlyPartial := r.Header.Get("HX-Request") == "true"
	if !onlyPartial {
		http.Redirect(w, r, "/admin/services", http.StatusSeeOther)
		return
	}

	component := editServiceForm(*service, *image)
	component.Render(r.Context(), w)
}

func (app *application) addEnvVarView(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	name := params.ByName("name")

	if name == "" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	component := addEnvVarForm()
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

	ports := app.dClient.GetContainerPorts(*service)
	if len(ports) == 0 {
		app.logger.Error("no ports found for container", "id", resp.ID)
		return
	}

	service.Port = &ports[0]
	err = app.services.UpdatePort(service.ID, *service.Port)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	envVars := app.dClient.GetContainerEnv(*service)
	err = app.services.UpdateEnvironmentVariables(service.ID, envVars)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	app.logger.Debug("started container", "id", resp.ID, "port", *service.Port)
	for _, host := range service.Hosts {
		app.proxy.Target[host] = service.Name
	}
}

/**
* Env Var Management
 */

func (app *application) updateService(w http.ResponseWriter, r *http.Request) {
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

	err = r.ParseForm()
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	/**
	 * TODO: currently only supports updating environment variables, make it not just do that lol
	 */

	envVars := make(map[string]string)
	for key, value := range r.PostForm {
		fmt.Println(key, value[0])
		if len(key) > 4 && key[:4] == "env-" {
			envVars[key[4:]] = value[0]
		} else if key == "new-env-key" {
			envVars[value[0]] = r.PostForm.Get("new-env-value")
		}
	}

	service.EnvironmentVariables = &envVars
	err = app.dClient.RemoveContainer(*service)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	err = app.services.UpdateStatus(service.ID, models.STOPPED)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	err = app.services.UpdateEnvironmentVariables(service.ID, envVars)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	image, err := app.images.Get(*service.ImageID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	go app.createContainerForService(service, image)
	onlyPartial := r.Header.Get("HX-Request") == "true"
	if !onlyPartial {
		http.Redirect(w, r, "/admin/services", http.StatusSeeOther)
		return
	}

	component := editServiceForm(*service, *image)
	component.Render(r.Context(), w)
}

func (app *application) deleteEnvVar(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	name := params.ByName("name")
	key := params.ByName("key")

	if name == "" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	if key == "" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	service, err := app.services.GetByName(name)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	err = r.ParseForm()
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	envVars := *service.EnvironmentVariables
	delete(envVars, key)

	err = app.dClient.RemoveContainer(*service)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	err = app.services.UpdateStatus(service.ID, models.STOPPED)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	err = app.services.UpdateEnvironmentVariables(service.ID, envVars)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	image, err := app.images.Get(*service.ImageID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	go app.createContainerForService(service, image)
	onlyPartial := r.Header.Get("HX-Request") == "true"
	if !onlyPartial {
		http.Redirect(w, r, "/admin/services", http.StatusSeeOther)
		return
	}

	component := editServiceForm(*service, *image)
	component.Render(r.Context(), w)
}
