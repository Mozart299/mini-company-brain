// Linear ingestion worker — polls the Linear GraphQL API for ticket state changes.
package linear

import (
	"context"
	"fmt"
	"time"

	"company-brain/pkg/types"
	"company-brain/queue"
)

type Config struct {
	APIKey string // Linear API key
	TeamID string // Linear team ID to filter tickets
}

type Worker struct {
	cfg    Config
	q      *queue.Client
	ticker *time.Ticker
}

func NewWorker(cfg Config, q *queue.Client) *Worker {
	return &Worker{cfg: cfg, q: q, ticker: time.NewTicker(60 * time.Second)}
}

// Run polls Linear every 60 seconds. Blocks until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	fmt.Printf("[linear] worker started for team %s\n", w.cfg.TeamID)
	for {
		select {
		case <-ctx.Done():
			w.ticker.Stop()
			return nil
		case <-w.ticker.C:
			if err := w.poll(ctx); err != nil {
				fmt.Printf("[linear] poll error: %v\n", err)
			}
		}
	}
}

func (w *Worker) poll(ctx context.Context) error {
	// TODO: POST to https://api.linear.app/graphql with query:
	// { issues(filter: { team: { id: { eq: $teamID } } }) { nodes { id title state { name } labels { nodes { name } } } } }
	// Authorization: Bearer {apiKey}
	event := types.Event{
		ID:        fmt.Sprintf("linear-%d", time.Now().UnixNano()),
		Kind:      types.EventLinearTicket,
		Source:    "linear",
		Timestamp: time.Now(),
		Payload: types.LinearTicket{
			ID:     "ENG-42",
			Title:  "Implement auth refresh flow",
			State:  "in_progress",
			Team:   w.cfg.TeamID,
			Labels: []string{"auth", "backend"},
		},
	}
	return w.q.Publish(ctx, event)
}
