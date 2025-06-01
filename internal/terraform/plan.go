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
	// Remove ANSI color codes
	plan = removeAnsiStringFromInput(plan)

	// Split into lines
	lines := strings.Split(plan, "\n")
	var cleanPlan strings.Builder

	// Find the start of the actual plan
	startIndex := -1
	for i, line := range lines {
		if strings.Contains(line, "Terraform will perform the following actions:") {
			startIndex = i
			break
		}
	}

	// If we found the start, capture from there
	if startIndex != -1 {
		// Skip the header line
		startIndex++

		// Capture until we hit the plan summary
		for i := startIndex; i < len(lines); i++ {
			line := lines[i]
			if strings.HasPrefix(strings.TrimSpace(line), "Plan: ") {
				cleanPlan.WriteString(line + "\n")
				break
			}

			// Format diff lines
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
				cleanPlan.WriteString(prefix + indentation + rest + "\n")
			} else {
				cleanPlan.WriteString(line + "\n")
			}
		}
	} else {
		// If we couldn't find the plan start, look for "No changes" message
		for _, line := range lines {
			if strings.Contains(line, "No changes. Infrastructure is up-to-date.") ||
				strings.Contains(line, "No changes. Your infrastructure matches the configuration.") {
				cleanPlan.WriteString(line + "\n")
				break
			}
		}
	}

	result := cleanPlan.String()

	// Truncate if too long
	if len(result) > maxlength {
		result = result[:maxlength]
	}

	return result
}
