package main

import (
	"net/http"

	"mc.jwoods.dev/ui"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.FS(ui.Files))
	mux.Handle("/static/", fileServer)

	mux.HandleFunc("/", app.home)

	return mux
}
