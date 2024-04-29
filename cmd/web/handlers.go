package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"nhooyr.io/websocket"
)

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
