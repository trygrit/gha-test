package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/trygrit/gha-terraform-commentor/internal/gh"
	"github.com/trygrit/gha-terraform-commentor/internal/terraform"

	"github.com/caarlos0/env/v11"
	"go.uber.org/zap"
)

// stripANSI removes ANSI color codes from a string
func stripANSI(s string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return ansiRegex.ReplaceAllString(s, "")
}

func main() {
	// Initialize logger with production config but writing to stdout
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"stdout"}
	logger, err := config.Build()
	if err != nil {
		fatalError("Failed to initialize logger:", err)
	}
	defer func() {
		// Ignore sync errors as they're not critical
		_ = logger.Sync()
	}()

	// Load configuration from environment
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		fatalError("Failed to parse environment variables:", err)
	}

	// Parse arguments
	var args = arguments
	_ = arg.MustParse(&args)

	logger.Debug("Parsed arguments", zap.String("command", args.Command), zap.String("directory", args.Directory))

	// Parse & validate the command
	commandParts := strings.Fields(args.Command)
	command := terraform.Command(commandParts[0])
	if !command.Validate() {
		fatalError("Invalid command provided. Valid commands are: fmt, plan, apply, destroy.", nil)
	}

	// Build command arguments
	cmdArgs := []string{"-chdir=" + args.Directory, string(command)}

	// Automatically add --auto-approve for apply commands
	if command == terraform.CommandApply {
		cmdArgs = append(cmdArgs, "--auto-approve")
	}

	// Add any additional arguments from the command string (excluding the first part which is the command)
	if len(commandParts) > 1 {
		cmdArgs = append(cmdArgs, commandParts[1:]...)
	}
	// Add any additional arguments from args.Args
	cmdArgs = append(cmdArgs, args.Args...)

	// Create a GitHub client
	client := gh.New(args.Token)

	ctx := context.Background()

	// Read the GitHub event data from stdin
	data, err := os.ReadFile("/github/workflow/event.json")
	if err != nil {
		fatalError("failed to read event file", err)
	}

	event, err := client.ParseEvent(data)
	if err != nil {
		fatalError("Error parsing GitHub event", err)
	}

	// Check if this is a PR
	if event.PullRequest.Number == 0 {
		logger.Debug("Not a PR, skipping comment post")
		// If it's not a PR, just run the terraform command and exit
		cmd := exec.Command("/usr/local/bin/terraform", cmdArgs...)
		// Set up pipes to capture output
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			fatalError("Failed to create stdout pipe:", err)
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			fatalError("Failed to create stderr pipe:", err)
		}

		// Start the command
		if err := cmd.Start(); err != nil {
			fatalError("Failed to start terraform command:", err)
		}

		// Stream output to stdout
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				fmt.Println(scanner.Text())
			}
		}()
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				fmt.Fprintln(os.Stderr, scanner.Text())
			}
		}()

		// Wait for command to complete
		if err := cmd.Wait(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				logger.Error("Terraform command failed",
					zap.String("command", string(command)),
					zap.Int("exit_code", exitErr.ExitCode()))
				os.Exit(exitErr.ExitCode())
			}
			fatalError("Failed to run terraform command:", err)
		}
		logger.Info("Terraform command completed successfully",
			zap.String("command", string(command)))
		os.Exit(0)
	}

	// Run terraform command
	cmd := exec.Command("/usr/local/bin/terraform", cmdArgs...)
	// Set up pipes to capture output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fatalError("Failed to create stdout pipe:", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		fatalError("Failed to create stderr pipe:", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		fatalError("Failed to start terraform command:", err)
	}

	// Stream output to stdout
	var output strings.Builder
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)
			output.WriteString(stripANSI(line) + "\n")
		}
	}()
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintln(os.Stderr, line)
			output.WriteString(stripANSI(line) + "\n")
		}
	}()

	// Wait for command to complete
	exitCode := "0"
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = fmt.Sprintf("%d", exitErr.ExitCode())
		}
	}

	searchPattern := fmt.Sprintf("### Terraform `%s`", args.Command)
	ok, err := client.DeleteExistingComment(ctx, event.Repository.Owner.Login, event.Repository.Name, event.PullRequest.Number, searchPattern)
	if err != nil {
		logger.Warn("Error checking for existing comments", zap.Error(err))
	}

	if ok {
		logger.Debug("Deleted existing comment")
	}

	// Create a comment body based on command
	comment, err := terraform.Comment(command, output.String(), exitCode, cfg.TerraformWorkspace, cfg.DetailsState)
	if err != nil {
		fatalError("Error creating comment body", err)
	}

	// Post comment to PR
	err = client.PostPRComment(ctx, event.Repository.Owner.Login, event.Repository.Name, event.PullRequest.Number, comment)
	if err != nil {
		logger.Error("Error posting comment", zap.Error(err))
		os.Exit(1) // Exit with error code 1 on failure
	}

	logger.Debug("Successfully posted comment to PR")
}

// fatalError logs the error message and exits the program with error code 1
func fatalError(s string, err error) {
	if len(s) > 0 {
		if !strings.HasSuffix(s, " ") {
			s += " "
		}
	}

	_, _ = fmt.Printf(s+"%v\n", err)
	os.Exit(1) // Exit with error code 1 on failure
}

// Config holds all configuration from environment variables
type Config struct {
	TerraformWorkspace string `env:"TERRAFORM_WORKSPACE" envDefault:"default"`
	DetailsState       string `env:"DETAILS_STATE" envDefault:"open"`
	Debug              bool   `env:"DEBUG" envDefault:"false"`
}

var arguments struct {
	Command   string   `arg:"positional, required" help:"Command run, fmt, plan, apply, etc."`
	Directory string   `arg:"positional, required" help:"Directory containing terraform files"`
	Token     string   `arg:"--token" help:"GitHub token for API access"`
	Args      []string `arg:"positional" help:"Additional arguments to pass to terraform command"`
}
