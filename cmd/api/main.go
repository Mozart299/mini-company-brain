package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"company-brain/api"
	"company-brain/coordinator"
	"company-brain/queue"
	"company-brain/store"
)

func main() {
	port := env("PORT", "8080")
	redisAddr := env("REDIS_ADDR", "localhost:6379")

	s := store.NewMemoryStore()
	d := coordinator.NewDetector(s)
	q := queue.NewClient(redisAddr)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Consume events from Redis stream and populate the in-memory store.
	consumer := store.NewConsumer(q, s)
	go func() {
		if err := consumer.Run(ctx); err != nil {
			fmt.Printf("[api] consumer error: %v\n", err)
		}
	}()

	srv := api.NewServer(s, d, port)
	fmt.Println("[api] starting server on :" + port)
	if err := srv.Run(ctx); err != nil {
		fmt.Printf("[api] server error: %v\n", err)
		os.Exit(1)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
