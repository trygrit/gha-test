package gh

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

type Client struct {
	gh *github.Client
}

func New(token string) *Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &Client{
		gh: github.NewClient(tc),
	}
}

func (c *Client) DeleteExistingComment(ctx context.Context, owner, repo string, prNumber int, searchPattern string) (bool, error) {
	comments, _, err := c.gh.Issues.ListComments(ctx, owner, repo, prNumber, nil)
	if err != nil {
		return false, err
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
	comment := &github.IssueComment{Body: github.String(body)}
	_, _, err := c.gh.Issues.CreateComment(ctx, owner, repo, prNumber, comment)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) ParseEvent(payload []byte) (Event, error) {
	var event Event

	err := json.Unmarshal(payload, &event)
	if err != nil {
		return event, err
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
