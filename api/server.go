// REST API server — exposes the knowledge store, drift alerts, and natural language queries.
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
	store        store.Store
	port         string
	anthropicKey string
}

func NewServer(s store.Store, port, anthropicKey string) *Server {
	return &Server{store: s, port: port, anthropicKey: anthropicKey}
}

func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /facts/{key}", s.handleGetFact)
	mux.HandleFunc("GET /facts", s.handleListFacts)
	mux.HandleFunc("GET /alerts", s.handleAlerts)
	mux.HandleFunc("POST /query", s.handleQuery)

	srv := &http.Server{Addr: ":" + s.port, Handler: cors(mux)}
	fmt.Printf("[api] listening on :%s\n", s.port)

	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	return srv.ListenAndServe()
}

// cors allows the Next.js dev server to call the API without a proxy.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
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
