package gh

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/go-github/v60/github"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

type Client struct {
	gh     *github.Client
	logger *zap.Logger
}

func New(token string) *Client {
	if token == "" {
		return &Client{
			logger: zap.NewNop(),
		}
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &Client{
		gh:     github.NewClient(tc),
		logger: zap.NewNop(),
	}
}

func (c *Client) DeleteExistingComment(ctx context.Context, owner, repo string, prNumber int, searchPattern string) (bool, error) {
	if c.gh == nil {
		return false, fmt.Errorf("GitHub client not initialized - token may be missing")
	}

	comments, _, err := c.gh.Issues.ListComments(ctx, owner, repo, prNumber, nil)
	if err != nil {
		return false, fmt.Errorf("failed to list comments: %w", err)
	}

	for _, comment := range comments {
		if comment.Body != nil && strings.Contains(*comment.Body, searchPattern) {
			_, err = c.gh.Issues.DeleteComment(ctx, owner, repo, *comment.ID)
			return err == nil, err
		}
	}

	return false, nil
}

func (c *Client) PostPRComment(ctx context.Context, owner, repo string, prNumber int, body string) error {
	if c.gh == nil {
		return fmt.Errorf("GitHub client not initialized - token may be missing")
	}

	comment := &github.IssueComment{Body: github.String(body)}
	_, _, err := c.gh.Issues.CreateComment(ctx, owner, repo, prNumber, comment)
	if err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}

	return nil
}

func (c *Client) ParseEvent(payload []byte) (Event, error) {
	var event Event

	err := json.Unmarshal(payload, &event)
	if err != nil {
		return event, fmt.Errorf("failed to parse event payload: %w", err)
	}

	return event, nil
}

// Event represents the structure of GitHub event payload
type Event struct {
	PullRequest PullRequest `json:"pull_request"`
	Repository  Repository  `json:"repository"`
}

type PullRequest struct {
	Number      int    `json:"number"`
	CommentsURL string `json:"comments_url"`
}

type Repository struct {
	Owner struct {
		Login string `json:"login"`
	} `json:"owner"`
	Name            string `json:"name"`
	IssueCommentURL string `json:"issue_comment_url"`
}
