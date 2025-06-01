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

	// Remove ANSI color codes
	plan = removeAnsiStringFromInput(plan)

	// Split into lines and process
	lines := strings.Split(plan, "\n")
	var cleanPlan strings.Builder
	started := false
	inStateLock := false
	inReading := false

	for _, line := range lines {
		// Handle state lock messages
		if strings.Contains(line, "Releasing state lock") {
			inStateLock = true
			continue
		}
		if inStateLock && strings.Contains(line, "Error:") {
			inStateLock = false
			continue
		}
		if inStateLock {
			continue
		}

		// Handle "Reading..." messages
		if strings.Contains(line, ": Reading...") {
			inReading = true
			continue
		}
		if inReading && strings.Contains(line, "Error:") {
			inReading = false
			continue
		}
		if inReading {
			continue
		}

		// Start capturing plan output
		if !started {
			for _, pattern := range planStartPatterns {
				if strings.Contains(line, pattern) {
					started = true
					cleanPlan.WriteString(line + "\n")
					break
				}
			}
			continue
		}

		// Stop at the end of the plan
		if started && strings.HasPrefix(strings.TrimSpace(line), "Plan: ") {
			cleanPlan.WriteString(line + "\n")
			break
		}

		// Format diff lines
		if started {
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
	}

	result := cleanPlan.String()

	// Truncate if too long
	if len(result) > maxlength {
		result = result[:maxlength]
	}

	return result
}
