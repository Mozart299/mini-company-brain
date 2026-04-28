// REST API server — exposes the knowledge store and drift alerts over HTTP.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"company-brain/pkg/types"
	"company-brain/store"
)

type Server struct {
	store store.Store
	port  string
}

func NewServer(s store.Store, port string) *Server {
	return &Server{store: s, port: port}
}

func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /facts/{key}", s.handleGetFact)
	mux.HandleFunc("GET /facts", s.handleListFacts)
	mux.HandleFunc("GET /alerts", s.handleAlerts)

	srv := &http.Server{Addr: ":" + s.port, Handler: mux}
	fmt.Printf("[api] listening on :%s\n", s.port)

	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	return srv.ListenAndServe()
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleGetFact(w http.ResponseWriter, r *http.Request) {
	fact, err := s.store.Get(r.Context(), r.PathValue("key"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fact)
}

func (s *Server) handleListFacts(w http.ResponseWriter, r *http.Request) {
	facts, err := s.store.List(r.Context(), r.URL.Query().Get("prefix"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if facts == nil {
		facts = []types.Fact{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(facts)
}

// handleAlerts reads drift alerts directly from the store (written by the coordinator).
func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	facts, err := s.store.List(r.Context(), "drift.alert:")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if facts == nil {
		facts = []types.Fact{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(facts)
}
