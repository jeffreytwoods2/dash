package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"mc.jwoods.dev/internal/models"
	"mc.jwoods.dev/internal/validator"
	"nhooyr.io/websocket"
)

type userSignupForm struct {
	Gamertag            string `form:"gamertag"`
	Password            string `form:"password"`
	Platform            string `form:"platform"`
	validator.Validator `form:"-"`
}

type userLoginForm struct {
	Gamertag            string `form:"gamertag"`
	Password            string `form:"password"`
	validator.Validator `form:"-"`
}

func (app *application) serviceWorkerStats(w http.ResponseWriter, r *http.Request) {
	env := envelope{
		"static_files": app.config.serviceWorker.staticFileList,
		"version":      version,
	}

	err := app.writeJSON(w, http.StatusOK, env, nil)
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	env := envelope{
		"status": "available",
		"system_info": map[string]string{
			"environment": app.config.env,
			"version":     version,
		},
	}

	err := app.writeJSON(w, http.StatusOK, env, nil)
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	playerList, err := getPlayerCoords()
	if err != nil {
		fmt.Println(err)
	}

	data := app.newTemplateData(r)

	data.Players = playerList

	app.render(w, r, http.StatusOK, "home.tmpl", data)
}

func (app *application) subscribeHandler(w http.ResponseWriter, r *http.Request) {
	err := app.subscribe(r.Context(), w, r)
	if errors.Is(err, context.Canceled) {
		return
	}
	if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
		websocket.CloseStatus(err) == websocket.StatusGoingAway {
		return
	}
	if err != nil {
		app.serverError(w, r, err)
		return
	}
}

// func (app *application) publishHandler(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != "POST" {
// 		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
// 		return
// 	}
// 	body := http.MaxBytesReader(w, r.Body, 8192)
// 	msg, err := io.ReadAll(body)
// 	if err != nil {
// 		http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
// 		return
// 	}

// 	app.publish(msg)

// 	w.WriteHeader(http.StatusAccepted)
// }

func (app *application) userSignup(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userSignupForm{}

	app.render(w, r, http.StatusOK, "signup.tmpl", data)
}

func (app *application) userSignupPost(w http.ResponseWriter, r *http.Request) {
	var form userSignupForm

	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	wl, err := getWhitelist()
	if err != nil {
		app.serverError(w, r, err)
	}

	form.CheckField(validator.NotBlank(form.Gamertag), "gamertag", "Gamertag must be provided")
	form.CheckField(validator.MaxChars(form.Gamertag, 16), "gamertag", "Gamertag cannot be more than 16 characters")
	form.CheckField(validator.GamertagFormat(form.Gamertag, form.Platform), "gamertag", "Bedrock must begin with a dot, and Java must not.")
	form.CheckField(validator.PermittedValue(form.Gamertag, wl...), "gamertag", "You are not whitelisted on this server. Did you select the correct platform?")
	form.CheckField(validator.MinChars(form.Password, 8), "password", "Password must be at least 8 characters long")
	form.CheckField(validator.NotBlank(form.Platform), "platform", "Platform must be provided")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "signup.tmpl", data)
		return
	}

	err = app.users.Insert(form.Gamertag, form.Password, form.Platform)
	if err != nil {
		if errors.Is(err, models.ErrDuplicateGamertag) {
			form.AddFieldError("gamertag", "Gamertag already registered")
			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusUnprocessableEntity, "signup.tmpl", data)
		} else {
			app.serverError(w, r, err)
		}

		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Signup successful! Please log in.")

	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}

func (app *application) userLogin(w http.ResponseWriter, r *http.Request) {
	data := app.newTemplateData(r)
	data.Form = userLoginForm{}
	app.render(w, r, http.StatusOK, "login.tmpl", data)
}

func (app *application) userLoginPost(w http.ResponseWriter, r *http.Request) {
	var form userLoginForm

	err := app.decodePostForm(r, &form)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Gamertag), "gamertag", "Please enter your gamertag")
	form.CheckField(validator.NotBlank(form.Password), "password", "Please enter your password")

	if !form.Valid() {
		data := app.newTemplateData(r)
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "login.tmpl", data)
		return
	}

	id, err := app.users.Authenticate(form.Gamertag, form.Password)
	if err != nil {
		if errors.Is(err, models.ErrInvalidCredentials) {
			form.AddNonFieldError("Gamertag or password is incorrect")

			data := app.newTemplateData(r)
			data.Form = form
			app.render(w, r, http.StatusUnprocessableEntity, "login.tmpl", data)
		} else {
			app.serverError(w, r, err)
		}
		return
	}

	err = app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), "authenticatedUserID", id)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {
	err := app.sessionManager.RenewToken(r.Context())
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Remove(r.Context(), "authenticatedUserID")
	app.sessionManager.Put(r.Context(), "flash", "Successfully logged out!")

	http.Redirect(w, r, "/user/login", http.StatusSeeOther)
}
