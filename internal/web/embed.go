package web

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
)

//go:embed static/*
var staticFS embed.FS

//go:embed templates/*
var templatesFS embed.FS

// GetStaticFS returns the embedded static files filesystem.
func GetStaticFS() http.FileSystem {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	return http.FS(sub)
}

// GetTemplates returns the parsed templates.
func GetTemplates() *template.Template {
	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		panic(err)
	}
	return tmpl
}
