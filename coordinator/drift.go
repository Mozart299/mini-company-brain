// Drift detection: compares planned work (Linear tickets) against actual work
// (GitHub commits/PRs) and raises alerts when they diverge.
// See docs/06-drift-detection.md.
package coordinator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"company-brain/pkg/types"
	"company-brain/store"
)

type Detector struct {
	store  store.Store
	alerts []types.DriftAlert
}

func NewDetector(s store.Store) *Detector {
	return &Detector{store: s}
}

// Run checks for drift every 5 minutes. Blocks until ctx is cancelled.
func (d *Detector) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	fmt.Println("[drift] detector started")
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

	// Simple heuristic: find commits that don't reference any ticket ID.
	ticketIDs := make(map[string]bool)
	for _, t := range tickets {
		if ticket, ok := t.Value.(types.LinearTicket); ok {
			ticketIDs[strings.ToLower(ticket.ID)] = true
		}
	}

	for _, c := range commits {
		commit, ok := c.Value.(types.GitHubCommit)
		if !ok {
			continue
		}
		msg := strings.ToLower(commit.Message)
		referenced := false
		for id := range ticketIDs {
			if strings.Contains(msg, strings.ToLower(id)) {
				referenced = true
				break
			}
		}
		if !referenced {
			alert := types.DriftAlert{
				ID:          fmt.Sprintf("drift-%d", time.Now().UnixNano()),
				Description: fmt.Sprintf("commit %s has no matching ticket", commit.SHA),
				Severity:    "medium",
				DetectedAt:  time.Now(),
				Evidence:    []string{commit.SHA, commit.Message},
			}
			d.alerts = append(d.alerts, alert)
			fmt.Printf("[drift] alert: %s\n", alert.Description)
		}
	}
}

// Alerts returns all active drift alerts.
func (d *Detector) Alerts() []types.DriftAlert {
	return d.alerts
}
