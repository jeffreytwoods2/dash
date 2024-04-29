package main

import (
	"net/http"

	"github.com/justinas/alice"
	"mc.jwoods.dev/ui"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.FS(ui.Files))
	mux.Handle("/static/", fileServer)

	mux.HandleFunc("/", app.home)
	mux.HandleFunc("/subscribe", app.subscribeHandler)

	standard := alice.New(app.recoverPanic, commonHeaders)
	return standard.Then(mux)
}
