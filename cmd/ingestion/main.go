package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"company-brain/ingestion/github"
	"company-brain/ingestion/linear"
	"company-brain/ingestion/slack"
	"company-brain/queue"
)

func main() {
	redisAddr := env("REDIS_ADDR", "localhost:6379")
	githubToken := env("GITHUB_TOKEN", "")
	githubRepo := env("GITHUB_REPO", "owner/repo")
	linearKey := env("LINEAR_API_KEY", "")
	linearTeam := env("LINEAR_TEAM_ID", "")

	q := queue.NewClient(redisAddr)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	errCh := make(chan error, 3)

	go func() {
		w := github.NewWorker(github.Config{Token: githubToken, Repo: githubRepo, Branch: "main"}, q)
		errCh <- w.Run(ctx)
	}()

	go func() {
		w := slack.NewWorker(q)
		errCh <- w.Run(ctx)
	}()

	go func() {
		w := linear.NewWorker(linear.Config{APIKey: linearKey, TeamID: linearTeam}, q)
		errCh <- w.Run(ctx)
	}()

	fmt.Println("[ingestion] all workers started")
	select {
	case <-ctx.Done():
		fmt.Println("[ingestion] shutting down")
	case err := <-errCh:
		fmt.Printf("[ingestion] worker error: %v\n", err)
		cancel()
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
