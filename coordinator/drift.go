// Drift detection: reads facts from the store, compares planned vs actual work,
// and writes DriftAlert facts back to the store under key "drift.alert:{id}".
// See docs/06-drift-detection.md.
package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"company-brain/pkg/types"
	"company-brain/store"
)

type Detector struct {
	store store.Store
}

func NewDetector(s store.Store) *Detector {
	return &Detector{store: s}
}

// Run checks for drift every 2 minutes. Blocks until ctx is cancelled.
func (d *Detector) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	fmt.Println("[drift] detector started")
	d.detect(ctx) // run immediately on start
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.detect(ctx)
		}
	}
}

func (d *Detector) detect(ctx context.Context) {
	commits, _ := d.store.List(ctx, "github.commit:")
	tickets, _ := d.store.List(ctx, "linear.ticket:")

	// Build a fast lookup: lowercase ticket ID → ticket
	ticketMap := make(map[string]types.LinearTicket)
	for _, t := range tickets {
		ticket := unmarshalAs[types.LinearTicket](t.Value)
		if ticket.ID != "" {
			ticketMap[strings.ToLower(ticket.ID)] = ticket
		}
	}

	// Pattern 1: commits with no ticket reference in the message.
	for _, c := range commits {
		commit := unmarshalAs[types.GitHubCommit](c.Value)
		if commit.SHA == "" {
			continue
		}
		msg := strings.ToLower(commit.Message)
		referenced := false
		for id := range ticketMap {
			if strings.Contains(msg, id) {
				referenced = true
				break
			}
		}
		if !referenced {
			sha := commit.SHA
			if len(sha) > 7 {
				sha = sha[:7]
			}
			d.raise(ctx, types.DriftAlert{
				ID:          "untracked-commit-" + sha,
				Description: fmt.Sprintf("commit %s (%q) references no ticket", sha, commit.Message),
				Severity:    "medium",
				DetectedAt:  time.Now(),
				Evidence:    []string{commit.SHA, commit.Message, commit.Repo},
			})
		}
	}

	// Pattern 2: tickets in_progress with no commit referencing them after 7 days.
	const staleAfter = 7 * 24 * time.Hour
	for _, t := range tickets {
		ticket := unmarshalAs[types.LinearTicket](t.Value)
		if ticket.State != "in_progress" || time.Since(t.CreatedAt) < staleAfter {
			continue
		}
		referenced := false
		for _, c := range commits {
			commit := unmarshalAs[types.GitHubCommit](c.Value)
			if strings.Contains(strings.ToLower(commit.Message), strings.ToLower(ticket.ID)) {
				referenced = true
				break
			}
		}
		if !referenced {
			days := int(time.Since(t.CreatedAt).Hours() / 24)
			d.raise(ctx, types.DriftAlert{
				ID:          "stale-ticket-" + ticket.ID,
				Description: fmt.Sprintf("ticket %s (%q) in_progress for %d days with no commits", ticket.ID, ticket.Title, days),
				Severity:    "high",
				DetectedAt:  time.Now(),
				Evidence:    []string{ticket.ID, ticket.Title, ticket.State},
			})
		}
	}

	fmt.Printf("[drift] cycle done — commits=%d tickets=%d\n", len(commits), len(tickets))
}

// raise persists a DriftAlert as a fact in the store.
// Idempotent: re-raising the same alert ID just updates the timestamp.
func (d *Detector) raise(ctx context.Context, alert types.DriftAlert) {
	var v any
	b, _ := json.Marshal(alert)
	json.Unmarshal(b, &v)

	if err := d.store.Set(ctx, types.Fact{
		Key:    "drift.alert:" + alert.ID,
		Value:  v,
		Source: "coordinator",
	}); err != nil {
		fmt.Printf("[drift] store alert failed: %v\n", err)
		return
	}
	fmt.Printf("[drift] [%s] %s\n", alert.Severity, alert.Description)
}

// unmarshalAs converts an any (typically map[string]any after JSON decode)
// into a typed struct via a round-trip through JSON.
func unmarshalAs[T any](v any) T {
	var out T
	b, _ := json.Marshal(v)
	json.Unmarshal(b, &out)
	return out
}
