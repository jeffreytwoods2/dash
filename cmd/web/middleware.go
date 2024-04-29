package main

import (
	"net/http"
	"time"
)

func (app *application) startBroadcast(next http.Handler) http.Handler {
	go func() {
		for {
			if len(app.subscribers) > 0 {
				time.Sleep(2 * time.Second)

				app.renderOnline()
			}
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
