package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"company-brain/coordinator"
	"company-brain/store"
)

func main() {
	redisAddr := env("REDIS_ADDR", "localhost:6379")
	storeNodes := env("STORE_NODES", "")

	var s store.Store
	if storeNodes != "" {
		ring := store.NewRing()
		rs := store.NewReplicatedStore(ring)
		for _, addr := range strings.Split(storeNodes, ",") {
			addr = strings.TrimSpace(addr)
			rs.RegisterNode(addr, store.NewNodeClient(addr))
		}
		s = rs
		fmt.Printf("[coordinator] distributed store: %s\n", storeNodes)
	} else {
		s = store.NewMemoryStore()
		fmt.Println("[coordinator] in-memory store (single-node mode)")
	}

	elector := coordinator.NewElector(redisAddr)
	detector := coordinator.NewDetector(s)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// detectorCancel lets us stop the detector when we lose leadership.
	var detectorCancel context.CancelFunc

	elector.Run(ctx,
		func() {
			var dCtx context.Context
			dCtx, detectorCancel = context.WithCancel(ctx)
			go detector.Run(dCtx)
		},
		func() {
			if detectorCancel != nil {
				detectorCancel()
				detectorCancel = nil
			}
			fmt.Println("[coordinator] stepped down — drift detection paused")
		},
	)
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
