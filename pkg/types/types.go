package types

import "time"

type EventKind string

const (
	EventGitHubCommit EventKind = "github.commit"
	EventGitHubPR     EventKind = "github.pull_request"
	EventGitHubIssue  EventKind = "github.issue"
	EventSlackMessage EventKind = "slack.message"
	EventLinearTicket EventKind = "linear.ticket"
)

// Event is a normalized unit of data published by an ingestion worker.
type Event struct {
	ID        string    `json:"id"`
	Kind      EventKind `json:"kind"`
	Source    string    `json:"source"`
	Timestamp time.Time `json:"timestamp"`
	Payload   any       `json:"payload"`
}

// Fact is a versioned piece of knowledge stored in the KV store.
type Fact struct {
	Key       string    `json:"key"`
	Value     any       `json:"value"`
	Source    string    `json:"source"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"created_at"`
}

// GitHubCommit is the payload for EventGitHubCommit.
type GitHubCommit struct {
	SHA     string `json:"sha"`
	Message string `json:"message"`
	Author  string `json:"author"`
	Repo    string `json:"repo"`
	Branch  string `json:"branch"`
}

// GitHubPR is the payload for EventGitHubPR.
type GitHubPR struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Author string `json:"author"`
	State  string `json:"state"` // open, closed, merged
	Repo   string `json:"repo"`
}

// LinearTicket is the payload for EventLinearTicket.
type LinearTicket struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	State  string `json:"state"` // backlog, todo, in_progress, done
	Team   string `json:"team"`
	Labels []string `json:"labels"`
}

// SlackMessage is the payload for EventSlackMessage.
type SlackMessage struct {
	Channel string `json:"channel"`
	User    string `json:"user"`
	Text    string `json:"text"`
}

// DriftAlert is raised by the coordinator when planned vs actual work diverges.
type DriftAlert struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"` // low, medium, high
	DetectedAt  time.Time `json:"detected_at"`
	Evidence    []string  `json:"evidence"`
}
