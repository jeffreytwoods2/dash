package main

import (
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"

	"github.com/justinas/nosurf"
	"mc.jwoods.dev/internal/models"
	"mc.jwoods.dev/ui"
)

type disc struct {
	Title     string
	Artist    string
	Namespace string
}

var discs = []disc{
	{
		Title:     "Island Girl",
		Artist:    "Surf Collective",
		Namespace: "islandgirlsurfcollectiveone",
	},
	{
		Title:     "The Gutter",
		Artist:    "Ice Cube",
		Namespace: "theguttericecubetwo",
	},
	{
		Title:     "Easy",
		Artist:    "Lionel Richie",
		Namespace: "easylionelrichiethree",
	},
}

type templateData struct {
	Players         []models.Player
	Discs           []disc
	Form            any
	Flash           string
	IsAuthenticated bool
	CSRFToken       string
}

func newTemplateCache() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	pages, err := fs.Glob(ui.Files, "html/pages/*.tmpl")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)

		patterns := []string{
			"html/base.tmpl",
			"html/partials/*.tmpl",
			page,
		}

		ts, err := template.New(name).ParseFS(ui.Files, patterns...)
		if err != nil {
			return nil, err
		}

		cache[name] = ts
	}

	return cache, nil
}

func (app *application) newTemplateData(r *http.Request) templateData {
	return templateData{
		Discs:           discs,
		Flash:           app.sessionManager.PopString(r.Context(), "flash"),
		IsAuthenticated: app.isAuthenticated(r),
		CSRFToken:       nosurf.Token(r),
	}
}
