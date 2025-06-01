package terraform

import (
	"regexp"
	"strings"
)

func removeAnsiStringFromInput(data string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)

	return ansiRegex.ReplaceAllString(string(data), "")
}

func ParsePlan(plan string, maxlength int) string {
	// Extract the plan section
	planStartPatterns := []string{
		"An execution plan has been generated and is shown below.",
		"Terraform used the selected providers to generate the following execution",
		"No changes. Infrastructure is up-to-date.",
		"No changes. Your infrastructure matches the configuration.",
		"Note: Objects have changed outside of Terraform",
	}

	plan = removeAnsiStringFromInput(plan)

	var cleanPlan string
	lines := strings.Split(plan, "\n")
	started := false
	for _, line := range lines {
		if !started {
			for _, pattern := range planStartPatterns {
				if strings.Contains(line, pattern) {
					started = true
					cleanPlan += line + "\n"
					break
				}
			}
		} else {
			cleanPlan += line + "\n"
			if strings.HasPrefix(line, "Plan: ") {
				break
			}
		}
	}

	// Format the diff lines
	var formattedLines []string
	for _, line := range strings.Split(cleanPlan, "\n") {
		trimmedLine := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmedLine, "-") ||
			strings.HasPrefix(trimmedLine, "+") ||
			strings.HasPrefix(trimmedLine, "~") {
			indentation := strings.Repeat(" ", len(line)-len(trimmedLine)-1)
			prefix := string(trimmedLine[0])
			if prefix == "~" {
				prefix = "!"
			}
			rest := trimmedLine[1:]
			formattedLines = append(formattedLines, prefix+indentation+rest)
		} else {
			formattedLines = append(formattedLines, line)
		}
	}

	result := strings.Join(formattedLines, "\n")

	// Truncate if too long
	if len(result) > maxlength {
		result = result[:maxlength]
	}

	return result
}
