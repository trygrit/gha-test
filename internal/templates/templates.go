package templates

import (
	"embed"
	"text/template"
)

//go:embed *.tmpl
var templates embed.FS

func LoadTemplates() (*template.Template, error) {
	// Parse all templates in the embedded filesystem
	tmpl, err := template.ParseFS(templates, "*.tmpl")
	if err != nil {
		return nil, err
	}
	return tmpl, nil
}
