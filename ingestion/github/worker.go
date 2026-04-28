// GitHub ingestion worker — polls the GitHub API for commits, PRs, and issues,
// then publishes normalized events to the message queue.
package github

import (
	"context"
	"fmt"
	"time"

	"company-brain/pkg/types"
	"company-brain/queue"
)

type Config struct {
	Token  string // GitHub personal access token
	Repo   string // e.g. "owner/repo"
	Branch string // e.g. "main"
}

type Worker struct {
	cfg    Config
	q      *queue.Client
	ticker *time.Ticker
}

func NewWorker(cfg Config, q *queue.Client) *Worker {
	return &Worker{cfg: cfg, q: q, ticker: time.NewTicker(30 * time.Second)}
}

// Run starts the poll loop. Blocks until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	fmt.Printf("[github] worker started for %s\n", w.cfg.Repo)
	for {
		select {
		case <-ctx.Done():
			w.ticker.Stop()
			return nil
		case <-w.ticker.C:
			if err := w.poll(ctx); err != nil {
				fmt.Printf("[github] poll error: %v\n", err)
			}
		}
	}
}

func (w *Worker) poll(ctx context.Context) error {
	// TODO: use net/http to call https://api.github.com/repos/{repo}/commits
	// Authorization: Bearer {token}
	// Parse response, build types.Event for each commit, publish to queue.
	event := types.Event{
		ID:        fmt.Sprintf("gh-%d", time.Now().UnixNano()),
		Kind:      types.EventGitHubCommit,
		Source:    "github",
		Timestamp: time.Now(),
		Payload: types.GitHubCommit{
			SHA:     "abc123",
			Message: "stub commit",
			Author:  "peter",
			Repo:    w.cfg.Repo,
			Branch:  w.cfg.Branch,
		},
	}
	return w.q.Publish(ctx, event)
}
