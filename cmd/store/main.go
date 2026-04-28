package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"company-brain/store"
)

func main() {
	nodeID := env("NODE_ID", "store-1")
	port := env("PORT", "7001")
	dataDir := env("DATA_DIR", "./data/"+nodeID)

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Printf("[store] mkdir %s: %v\n", dataDir, err)
		os.Exit(1)
	}

	s, err := store.NewBadgerStore(dataDir)
	if err != nil {
		fmt.Printf("[store] open badger: %v\n", err)
		os.Exit(1)
	}
	defer s.Close()

	srv := &http.Server{Addr: ":" + port, Handler: store.NewNodeServer(s)}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	fmt.Printf("[store] node %s listening on :%s (data: %s)\n", nodeID, port, dataDir)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("[store] error: %v\n", err)
		os.Exit(1)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
