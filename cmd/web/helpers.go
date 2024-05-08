package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"math"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/go-playground/form/v4"
	"mc.jwoods.dev/internal/models"
	"mc.jwoods.dev/ui"
	"nhooyr.io/websocket"
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

func (app *application) clientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
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

func (app *application) renderOnline() {
	players, err := getPlayerCoords()
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	data := templateData{
		Players: players,
	}

	ts, err := template.New("players").ParseFS(ui.Files, "html/partials/players.tmpl")
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	buf := new(bytes.Buffer)

	err = ts.ExecuteTemplate(buf, "players", data)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	app.publish(buf.Bytes())
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

func (app *application) subscribe(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var (
		mu     sync.Mutex
		c      *websocket.Conn
		closed bool
	)

	s := &subscriber{
		msgs: make(chan []byte, app.subscriberMessageBuffer),
		closeSlow: func() {
			mu.Lock()
			defer mu.Unlock()
			closed = true
			if c != nil {
				c.Close(websocket.StatusPolicyViolation, "connection too slow to keep up with messages")
			}
		},
	}

	app.addSubcriber(s)
	defer app.deleteSubscriber(s)

	c2, err := websocket.Accept(w, r, nil)
	if err != nil {
		return err
	}
	mu.Lock()
	if closed {
		mu.Unlock()
		return net.ErrClosed
	}

	c = c2
	mu.Unlock()
	defer c.CloseNow()

	ctx = c.CloseRead(ctx)

	for {
		select {
		case msg := <-s.msgs:
			err := writeTimeout(ctx, 5*time.Second, c, msg)
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (app *application) publish(msg []byte) {
	app.subscribersMu.Lock()
	defer app.subscribersMu.Unlock()

	for s := range app.subscribers {
		select {
		case s.msgs <- msg:
		default:
			go s.closeSlow()
		}
	}
}

func (app *application) addSubcriber(s *subscriber) {
	app.subscribersMu.Lock()
	app.subscribers[s] = struct{}{}
	app.subscribersMu.Unlock()
}

func (app *application) deleteSubscriber(s *subscriber) {
	app.subscribersMu.Lock()
	delete(app.subscribers, s)
	app.subscribersMu.Unlock()
}

func writeTimeout(ctx context.Context, timeout time.Duration, c *websocket.Conn, msg []byte) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return c.Write(ctx, websocket.MessageText, msg)
}

func (app *application) decodePostForm(r *http.Request, dst any) error {
	err := r.ParseForm()
	if err != nil {
		return err
	}

	err = app.formDecoder.Decode(dst, r.PostForm)
	if err != nil {
		var invalidDecoderError *form.InvalidDecoderError

		if errors.As(err, &invalidDecoderError) {
			panic(err)
		}

		return err
	}

	return nil
}
