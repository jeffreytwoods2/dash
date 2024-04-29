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

	dynamic := alice.New(app.sessionManager.LoadAndSave)

	mux.Handle("GET /{$}", dynamic.ThenFunc(app.home))
	mux.Handle("GET /subscribe", dynamic.ThenFunc(app.subscribeHandler))
	mux.Handle("GET /user/signup", dynamic.ThenFunc(app.userSignup))
	mux.Handle("POST /user/signup", dynamic.ThenFunc(app.userSignupPost))
	mux.Handle("GET /user/login", dynamic.ThenFunc(app.userLogin))
	mux.Handle("POST /user/login", dynamic.ThenFunc(app.userLoginPost))
	mux.Handle("POST /user/logout", dynamic.ThenFunc(app.userLogoutPost))

	standard := alice.New(app.recoverPanic, commonHeaders)
	return standard.Then(mux)
}
