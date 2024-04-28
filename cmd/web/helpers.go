package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"

	"mc.jwoods.dev/internal/models"
)

type OnlineRes struct {
	OnlinePlayers int      `json:"onlinePlayers"`
	Online        []string `json:"online"`
}

type PlayerLocationRes struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type WorldRes struct {
	World string `json:"world"`
}

func (app *application) serverError(w http.ResponseWriter, r *http.Request, err error) {
	var (
		method = r.Method
		uri    = r.URL.RequestURI()
	)

	app.logger.Error(err.Error(), "method", method, "uri", uri)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (app *application) render(w http.ResponseWriter, r *http.Request, status int, page string, data templateData) {
	ts, ok := app.templateCache[page]
	if !ok {
		err := fmt.Errorf("the template %s does not exist", page)
		app.serverError(w, r, err)
		return
	}

	buf := new(bytes.Buffer)

	err := ts.ExecuteTemplate(buf, "base", data)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	w.WriteHeader(status)

	buf.WriteTo(w)
}

func sendRcon(endpoint string) ([]byte, error) {
	url := fmt.Sprintf("http://mc.jwoods.dev:8000/api/v1/%s", endpoint)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", "ODk1NDE0MzgwNDI5MDc0MjczMDA")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func getPlayers() ([]string, error) {
	getPlayers, err := sendRcon("players/online")
	if err != nil {
		return nil, err
	}
	var target OnlineRes
	err = json.Unmarshal(getPlayers, &target)
	if err != nil {
		return nil, err
	}
	return target.Online, nil
}

func getPlayerCoords() ([]models.Player, error) {
	var players []models.Player
	playerList, err := getPlayers()
	if err != nil {
		return nil, err
	}

	for _, player := range playerList {
		var currPlayer models.Player
		endpoint := fmt.Sprintf("players/%s/location", player)
		rawLocation, err := sendRcon(endpoint)
		if err != nil {
			return nil, err
		}

		endpoint = fmt.Sprintf("players/%s/world", player)
		rawWorld, err := sendRcon(endpoint)
		if err != nil {
			return nil, err
		}

		var locationRes PlayerLocationRes
		err = json.Unmarshal(rawLocation, &locationRes)
		if err != nil {
			return nil, err
		}

		var worldRes WorldRes
		err = json.Unmarshal(rawWorld, &worldRes)
		if err != nil {
			return nil, err
		}

		var world string
		switch worldRes.World {
		case "world":
			world = "Overworld"
		case "world_nether":
			world = "Nether"
		case "world_the_end":
			world = "The End"
		}

		locationStr := fmt.Sprintf("%s (%d, %d, %d)", world, int(math.Round(locationRes.X)), int(math.Round(locationRes.Y)), int(math.Round(locationRes.Z)))
		currPlayer.Name = player
		currPlayer.Location = locationStr
		players = append(players, currPlayer)
	}
	return players, nil
}