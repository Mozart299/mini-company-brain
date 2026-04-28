package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"company-brain/api"
	"company-brain/coordinator"
	"company-brain/queue"
	"company-brain/store"
)

func main() {
	port := env("PORT", "8080")
	redisAddr := env("REDIS_ADDR", "localhost:6379")
	storeNodes := env("STORE_NODES", "") // e.g. "localhost:7001,localhost:7002,localhost:7003"

	var s store.Store
	if storeNodes != "" {
		ring := store.NewRing()
		rs := store.NewReplicatedStore(ring)
		for _, addr := range strings.Split(storeNodes, ",") {
			addr = strings.TrimSpace(addr)
			rs.RegisterNode(addr, store.NewNodeClient(addr))
		}
		s = rs
		fmt.Printf("[api] distributed store: %s\n", storeNodes)
	} else {
		s = store.NewMemoryStore()
		fmt.Println("[api] in-memory store (single-node mode)")
	}

	d := coordinator.NewDetector(s)
	q := queue.NewClient(redisAddr)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

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
