package web

import (
	"embed"
	"html/template"
	"io"
	"net/http"
)

//go:embed static/*
var StaticFS embed.FS

//go:embed templates/*
var templateFS embed.FS

var (
	statusTpl *template.Template
	adminTpl  *template.Template
)

func init() {
	statusTpl = template.Must(template.ParseFS(templateFS, "templates/status.html"))
	adminTpl = template.Must(template.ParseFS(templateFS, "templates/admin.html"))
}

func RenderStatusPage(w io.Writer, data interface{}) {
	statusTpl.Execute(w, data)
}

func RenderAdminPage(w io.Writer, data interface{}) {
	adminTpl.Execute(w, data)
}
