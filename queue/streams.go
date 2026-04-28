// Package queue wraps Redis Streams for event publishing and consumption.
// See docs/02-redis-streams.md for a full explanation of how this works.
package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"company-brain/pkg/types"

	"github.com/redis/go-redis/v9"
)

const defaultStream = "company-brain:events"

type Client struct {
	rdb    *redis.Client
	stream string
}

func NewClient(redisAddr string) *Client {
	return &Client{
		rdb:    redis.NewClient(&redis.Options{Addr: redisAddr}),
		stream: defaultStream,
	}
}

// Publish serializes an event and appends it to the Redis Stream.
func (c *Client) Publish(ctx context.Context, event types.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	if err := c.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: c.stream,
		Values: map[string]any{"data": string(data)},
	}).Err(); err != nil {
		return fmt.Errorf("xadd: %w", err)
	}
	fmt.Printf("[queue] published %s (id=%s)\n", event.Kind, event.ID)
	return nil
}

// Consume reads events from the stream via a consumer group and calls handler for each.
// Blocks until ctx is cancelled. Acks messages only after handler succeeds.
func (c *Client) Consume(ctx context.Context, group, consumer string, handler func(types.Event) error) error {
	// Create group from the beginning of the stream ("0"); ignore error if already exists.
	if err := c.rdb.XGroupCreateMkStream(ctx, c.stream, group, "0").Err(); err != nil {
		if !strings.Contains(err.Error(), "BUSYGROUP") {
			return fmt.Errorf("create consumer group: %w", err)
		}
	}

	fmt.Printf("[queue] consumer %s/%s ready\n", group, consumer)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		streams, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    group,
			Consumer: consumer,
			Streams:  []string{c.stream, ">"},
			Count:    10,
			Block:    2 * time.Second,
		}).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}
			if ctx.Err() != nil {
				return nil
			}
			fmt.Printf("[queue] xreadgroup error: %v\n", err)
			continue
		}

		for _, s := range streams {
			for _, msg := range s.Messages {
				raw, _ := msg.Values["data"].(string)
				var event types.Event
				if err := json.Unmarshal([]byte(raw), &event); err != nil {
					fmt.Printf("[queue] unmarshal error: %v\n", err)
					c.rdb.XAck(ctx, c.stream, group, msg.ID)
					continue
				}
				if err := handler(event); err != nil {
					fmt.Printf("[queue] handler error: %v\n", err)
					continue
				}
				c.rdb.XAck(ctx, c.stream, group, msg.ID)
			}
		}
	}
}
