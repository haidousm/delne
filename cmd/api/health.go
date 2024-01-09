package main

import "net/http"

func (app *application) health(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}
