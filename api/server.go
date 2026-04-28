// REST API server — exposes the knowledge store and drift alerts over HTTP.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"company-brain/coordinator"
	"company-brain/store"
)

type Server struct {
	store    store.Store
	detector *coordinator.Detector
	port     string
}

func NewServer(s store.Store, d *coordinator.Detector, port string) *Server {
	return &Server{store: s, detector: d, port: port}
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
	key := r.PathValue("key")
	fact, err := s.store.Get(r.Context(), key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fact)
}

func (s *Server) handleListFacts(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("prefix")
	facts, err := s.store.List(r.Context(), prefix)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(facts)
}

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.detector.Alerts())
}
