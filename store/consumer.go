// Consumer reads events from the queue and writes facts into the knowledge store.
package store

import (
	"context"
	"encoding/json"
	"fmt"

	"company-brain/pkg/types"
	"company-brain/queue"
)

type Consumer struct {
	q *queue.Client
	s Store
}

func NewConsumer(q *queue.Client, s Store) *Consumer {
	return &Consumer{q: q, s: s}
}

// Run subscribes to the stream and stores every incoming event as a Fact.
func (c *Consumer) Run(ctx context.Context) error {
	return c.q.Consume(ctx, "store-workers", "api-1", func(event types.Event) error {
		key := factKey(event)
		fmt.Printf("[store] storing %s\n", key)
		return c.s.Set(ctx, types.Fact{
			Key:    key,
			Value:  event.Payload,
			Source: event.Source,
		})
	})
}

func factKey(event types.Event) string {
	p := toMap(event.Payload)
	switch event.Kind {
	case types.EventGitHubCommit:
		if sha, _ := p["sha"].(string); sha != "" {
			return "github.commit:" + sha
		}
	case types.EventGitHubPR:
		if num, _ := p["number"].(float64); num != 0 {
			return fmt.Sprintf("github.pr:%d", int(num))
		}
	case types.EventLinearTicket:
		if id, _ := p["id"].(string); id != "" {
			return "linear.ticket:" + id
		}
	case types.EventSlackMessage:
		return "slack.message:" + event.ID
	}
	return string(event.Kind) + ":" + event.ID
}

// toMap normalizes event.Payload to map[string]any regardless of whether it
// arrived as a typed struct or was decoded from JSON into an interface{}.
func toMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	b, _ := json.Marshal(v)
	var m map[string]any
	json.Unmarshal(b, &m)
	return m
}
