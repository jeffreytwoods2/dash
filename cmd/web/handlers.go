package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"mc.jwoods.dev/internal/validator"
	"nhooyr.io/websocket"
)

type userSignupForm struct {
	Gamertag string
	Password string
	Platform string
	validator.Validator
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	playerList, err := getPlayerCoords()
	if err != nil {
		fmt.Println(err)
	}

	data := templateData{
		Players: playerList,
	}

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
	data := app.newTemplateData()
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

	form.CheckField(validator.NotBlank(form.Gamertag), "gamertag", "Gamertag must be provided")
	form.CheckField(validator.MaxChars(form.Gamertag, 16), "gamertag", "Gamertag cannot be more than 16 characters")
	form.CheckField(validator.NotBlank(form.Platform), "platform", "Platform must be provided")

	if !form.Valid() {
		data := app.newTemplateData()
		data.Form = form
		app.render(w, r, http.StatusUnprocessableEntity, "signup.tmpl", data)
		return
	}
}

func (app *application) userLogin(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Display a form for logging in a user...")
}

func (app *application) userLoginPost(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Authenticate and login the user...")
}

func (app *application) userLogoutPost(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Logout the user...")
}
