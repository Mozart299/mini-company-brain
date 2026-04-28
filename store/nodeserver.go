// NodeServer exposes a Store over HTTP so remote processes can read/write it.
// Each store node runs one of these; the ReplicatedStore talks to all of them.
package store

import (
	"encoding/json"
	"net/http"

	"company-brain/pkg/types"
)

type NodeServer struct {
	store Store
	mux   *http.ServeMux
}

func NewNodeServer(s Store) *NodeServer {
	ns := &NodeServer{store: s, mux: http.NewServeMux()}
	ns.mux.HandleFunc("PUT /facts/{key}", ns.handleSet)
	ns.mux.HandleFunc("GET /facts/{key}", ns.handleGet)
	ns.mux.HandleFunc("DELETE /facts/{key}", ns.handleDelete)
	ns.mux.HandleFunc("GET /facts", ns.handleList)
	ns.mux.HandleFunc("GET /health", ns.handleHealth)
	return ns
}

func (ns *NodeServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ns.mux.ServeHTTP(w, r)
}

func (ns *NodeServer) handleSet(w http.ResponseWriter, r *http.Request) {
	var fact types.Fact
	if err := json.NewDecoder(r.Body).Decode(&fact); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fact.Key = r.PathValue("key")
	if err := ns.store.Set(r.Context(), fact); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (ns *NodeServer) handleGet(w http.ResponseWriter, r *http.Request) {
	fact, err := ns.store.Get(r.Context(), r.PathValue("key"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fact)
}

func (ns *NodeServer) handleDelete(w http.ResponseWriter, r *http.Request) {
	if err := ns.store.Delete(r.Context(), r.PathValue("key")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (ns *NodeServer) handleList(w http.ResponseWriter, r *http.Request) {
	facts, err := ns.store.List(r.Context(), r.URL.Query().Get("prefix"))
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

func (ns *NodeServer) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
