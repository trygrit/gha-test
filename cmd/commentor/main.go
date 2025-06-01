package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/trygrit/gha-terraform-commentor/internal/gh"
	"github.com/trygrit/gha-terraform-commentor/internal/terraform"

	"github.com/caarlos0/env/v11"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger := zap.Must(zap.NewProduction())
	defer func() {
		err := logger.Sync()
		if err != nil {
			fatalError("Failed to sync logger:", err)
		}
	}()

	// Load configuration from environment
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		fatalError("", err)
	}

	// Parse arguments
	var args = arguments
	_ = arg.MustParse(&args)

	logger.Debug("Parsed arguments", zap.String("command", args.Command), zap.String("source_file", args.Input), zap.String("exit_code", args.CommandExitCode))

	// Parse & validate the command
	command := terraform.Command(args.Command)
	if !command.Validate() {
		fatalError("Invalid command provided. Valid commands are: fmt, plan, apply, destroy.", nil)
	}

	// Create a GitHub client
	client := gh.New(cfg.GitHubToken)

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
		os.Exit(0)
	}

	// Read the Terraform plan output file
	plan, err := os.ReadFile(args.Input)
	if err != nil {
		fatalError("failed to read terraform plan output", err)
	}

	searchPattern := fmt.Sprintf("### Terraform `%s`", args.Command)
	ok, err := client.DeleteExistingComment(ctx, event.Repository.Owner, event.Repository.Name, event.PullRequest.Number, searchPattern)
	if err != nil {
		logger.Warn("Error checking for existing comments", zap.Error(err))
	}

	if ok {
		logger.Debug("Deleted existing comment")
	}

	// Create a comment body based on command
	comment, err := terraform.Comment(command, string(plan), args.CommandExitCode, cfg.TerraformWorkspace, cfg.DetailsState)
	if err != nil {
		fatalError("Error creating comment body", err)
	}

	// Post comment to PR
	err = client.PostPRComment(ctx, event.Repository.Owner, event.Repository.Name, event.PullRequest.Number, comment)
	if err != nil {
		fatalError("Error posting comment", err)
	}

	logger.Debug("Successfully posted comment to PR")
}

// fatalError logs the error message and exits the program, appending a newline
func fatalError(s string, err error) {
	if len(s) > 0 {
		if !strings.HasSuffix(s, " ") {
			s += s + " "
		}
	}

	_, _ = fmt.Printf(s+"%v\n", err)
	os.Exit(0)
}

// Config holds all configuration from environment variables
type Config struct {
	GitHubToken        string `env:"GITHUB_TOKEN,required"`
	TerraformWorkspace string `env:"TERRAFORM_WORKSPACE" envDefault:"default"`
	DetailsState       string `env:"DETAILS_STATE" envDefault:"open"`
	Debug              bool   `env:"DEBUG" envDefault:"false"`
}

var arguments struct {
	Command         string `arg:"positional, required" help:"Command run, fmt, plan, apply, etc."`
	Input           string `arg:"positional, required" help:"Path to input file to parse"`
	CommandExitCode string `arg:"positional, required" help:"Command exit code"`
}
