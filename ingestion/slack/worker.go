// Slack ingestion worker — uses a mock feed in Milestone 1.
// Replace mockMessages with a real Slack Events API or Socket Mode client later.
package slack

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"company-brain/pkg/types"
	"company-brain/queue"
)

var mockMessages = []string{
	"shipped the new auth flow",
	"PR for dashboard is ready for review",
	"hotfix deployed to prod",
	"weekly sync notes: decided to drop feature X",
	"@team the API is down, investigating",
}

type Worker struct {
	q      *queue.Client
	ticker *time.Ticker
}

func NewWorker(q *queue.Client) *Worker {
	return &Worker{q: q, ticker: time.NewTicker(15 * time.Second)}
}

// Run emits a mock Slack message every 15 seconds. Blocks until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	fmt.Println("[slack] mock worker started")
	for {
		select {
		case <-ctx.Done():
			w.ticker.Stop()
			return nil
		case <-w.ticker.C:
			msg := mockMessages[rand.Intn(len(mockMessages))]
			event := types.Event{
				ID:        fmt.Sprintf("slack-%d", time.Now().UnixNano()),
				Kind:      types.EventSlackMessage,
				Source:    "slack",
				Timestamp: time.Now(),
				Payload: types.SlackMessage{
					Channel: "#engineering",
					User:    "peter",
					Text:    msg,
				},
			}
			if err := w.q.Publish(ctx, event); err != nil {
				fmt.Printf("[slack] publish error: %v\n", err)
			}
		}
	}
}
