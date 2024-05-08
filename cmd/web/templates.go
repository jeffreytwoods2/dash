package main

import (
	"html/template"
	"io/fs"
	"path/filepath"

	"mc.jwoods.dev/internal/models"
	"mc.jwoods.dev/ui"
)

type templateData struct {
	Players []models.Player
	Form    any
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

func (app *application) newTemplateData() templateData {
	return templateData{}
}
