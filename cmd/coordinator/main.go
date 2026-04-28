package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"company-brain/coordinator"
	"company-brain/store"
)

func main() {
	redisAddr := env("REDIS_ADDR", "localhost:6379")

	s := store.NewMemoryStore()
	elector := coordinator.NewElector(redisAddr)
	detector := coordinator.NewDetector(s)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Println("[coordinator] starting")
	elector.Run(ctx,
		func() { go detector.Run(ctx) },
		func() { fmt.Println("[coordinator] stepped down, pausing drift detection") },
	)
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
