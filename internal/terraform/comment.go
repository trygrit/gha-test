package terraform

import (
	"fmt"
	"strings"

	"github.com/trygrit/gha-terraform-commentor/internal/templates"
)

func Comment(command Command, input, exitCode, workspace, detailsState string) (string, error) {
	const (
		maxCommentLength = 65536 // Max length for GitHub comments
	)

	tmpl, err := templates.LoadTemplates()
	if err != nil {
		return "", err
	}

	// Determine which template to use
	var tmplName string

	switch command {
	case CommandPlan:
		if exitCode == "0" || exitCode == "2" {
			// Process successful plan output
			input = ParsePlan(input, maxCommentLength)
			tmplName = "plan_success.tmpl"
		} else {
			tmplName = "plan_failure.tmpl"
		}
	case CommandApply:
		if exitCode == "0" {
			// For apply, we want to show the full output but clean it up
			input = strings.TrimSpace(input)
			tmplName = "apply_success.tmpl"
		} else {
			tmplName = "apply_failure.tmpl"
		}
	default:
		tmplName = "general.tmpl"
	}

	if tmpl.Lookup(tmplName) == nil {
		return "", fmt.Errorf("template %s not found", tmplName)
	}

	// Execute template with data
	data := map[string]interface{}{
		"Command":      command,
		"Input":        input,
		"Workspace":    workspace,
		"DetailsState": detailsState,
	}

	var buf strings.Builder
	if err := tmpl.Lookup(tmplName).Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error executing template: %v", err)
	}

	return buf.String(), nil
}
