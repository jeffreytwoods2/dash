package main

import (
	"net/http"

	"github.com/justinas/alice"
	"mc.jwoods.dev/ui"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.FS(ui.Files))

	mux.Handle("GET /static/", fileServer)
	mux.HandleFunc("GET /healthcheck", app.healthcheckHandler)
	mux.HandleFunc("GET /sw", app.serviceWorkerStats)
	mux.HandleFunc("POST /givedisc", app.giveDisc)

	// TODO: Handle this in Caddy or Cloudflare instead in the future
	mux.HandleFunc("GET /serviceworker", app.serviceWorker)

	dynamic := alice.New(app.sessionManager.LoadAndSave, noSurf)

	mux.Handle("GET /{$}", dynamic.ThenFunc(app.home))
	mux.Handle("GET /user/signup", dynamic.ThenFunc(app.userSignup))
	mux.Handle("POST /user/signup", dynamic.ThenFunc(app.userSignupPost))
	mux.Handle("GET /user/login", dynamic.ThenFunc(app.userLogin))
	mux.Handle("POST /user/login", dynamic.ThenFunc(app.userLoginPost))

	protected := dynamic.Append(app.requireAuthentication)

	mux.Handle("GET /subscribe", protected.ThenFunc(app.subscribeHandler))
	mux.Handle("POST /user/logout", protected.ThenFunc(app.userLogoutPost))
	// mux.Handle("POST /givedisc", protected.ThenFunc(app.giveDisc))

	standard := alice.New(app.recoverPanic, commonHeaders)
	return standard.Then(mux)
}
