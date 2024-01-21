package main

import (
	"fmt"
	"net/http"
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
