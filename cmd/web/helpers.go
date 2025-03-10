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
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-playground/form/v4"
	"mc.jwoods.dev/internal/models"
	"mc.jwoods.dev/ui"
	"nhooyr.io/websocket"
)

type subscriber struct {
	msgs      chan []byte
	closeSlow func()
}

type OnlineRes struct {
	Players []string `json:"players"`
}

type PlayerLocationRes struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type WorldRes struct {
	World string `json:"world"`
}

type envelope map[string]any

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

func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	js = append(js, '\n')

	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
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
	players, err := app.getPlayerCoords()
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

func (app *application) sendAPIReq(endpoint string) ([]byte, error) {
	url := fmt.Sprintf("http://localhost:8000/api/%s", endpoint)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+app.config.jwt)
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

func (app *application) getPlayers() ([]string, error) {
	getPlayers, err := app.sendAPIReq("players/online")
	if err != nil {
		return nil, err
	}
	var target OnlineRes
	err = json.Unmarshal(getPlayers, &target)
	if err != nil {
		return nil, err
	}
	return target.Players, nil
}

func (app *application) getPlayerCoords() ([]models.Player, error) {
	var players []models.Player
	playerList, err := app.getPlayers()
	if err != nil {
		return nil, err
	}

	for _, player := range playerList {
		var currPlayer models.Player
		endpoint := fmt.Sprintf("players/%s/location", player)
		rawLocation, err := app.sendAPIReq(endpoint)
		if err != nil {
			return nil, err
		}

		endpoint = fmt.Sprintf("players/%s", player)
		rawWorld, err := app.sendAPIReq(endpoint)
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
		msgs: make(chan []byte, app.config.subscriberMessageBuffer),
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

func (app *application) isAuthenticated(r *http.Request) bool {
	return app.sessionManager.Exists(r.Context(), "authenticatedUserID")
}

func (app *application) playerIsWhitelisted(player string) (bool, error) {
	type whitelist struct {
		Whitelisted bool `json:"whitelisted"`
	}
	wl := whitelist{}
	endpoint := fmt.Sprintf("whitelist/players/%s", player)
	result, err := app.sendAPIReq(endpoint)
	if err != nil {
		return false, err
	}

	err = json.Unmarshal(result, &wl)
	if err != nil {
		return false, err
	}

	return wl.Whitelisted, nil
}

func (cfg *config) buildStaticFileList() error {
	len_path_prefix := len(cfg.serviceWorker.staticDir)
	err := filepath.Walk(cfg.serviceWorker.staticDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				short_filepath := path[len_path_prefix:]
				uri := fmt.Sprintf("https://dash.jwoods.dev/static%s", short_filepath)
				cfg.serviceWorker.staticFileList = append(cfg.serviceWorker.staticFileList, uri)
			}
			return nil
		})

	return err
}
