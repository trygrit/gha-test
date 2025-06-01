package templates

import (
	"embed"
	"fmt"
	"text/template"
)

//go:embed *.tmpl
var templates embed.FS

func LoadTemplates() (*template.Template, error) {
	// Parse all templates in the embedded filesystem
	tmpl := template.New("")

	// List of required templates
	requiredTemplates := []string{"plan_success.tmpl", "plan_failure.tmpl", "general.tmpl"}

	for _, name := range requiredTemplates {
		content, err := templates.ReadFile(name)
		if err != nil {
			return nil, fmt.Errorf("failed to read template %s: %v", name, err)
		}

		_, err = tmpl.New(name).Parse(string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %s: %v", name, err)
		}
	}

	return tmpl, nil
}
