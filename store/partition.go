// Consistent hashing: maps keys to store nodes so adding/removing
// a node only remaps ~1/N of keys instead of all of them.
// See docs/03-consistent-hashing.md for the full explanation.
package store

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"sync"
)

const virtualNodes = 150 // replicas per physical node on the ring

type Ring struct {
	mu      sync.RWMutex
	nodes   map[uint32]string // hash -> node address
	sorted  []uint32          // sorted hash keys for binary search
}

func NewRing() *Ring {
	return &Ring{nodes: make(map[uint32]string)}
}

// AddNode places a physical node on the ring using virtualNodes virtual points.
func (r *Ring) AddNode(addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := 0; i < virtualNodes; i++ {
		h := hash(fmt.Sprintf("%s#%d", addr, i))
		r.nodes[h] = addr
		r.sorted = append(r.sorted, h)
	}
	sort.Slice(r.sorted, func(i, j int) bool { return r.sorted[i] < r.sorted[j] })
}

// RemoveNode removes all virtual points for a physical node.
func (r *Ring) RemoveNode(addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := 0; i < virtualNodes; i++ {
		h := hash(fmt.Sprintf("%s#%d", addr, i))
		delete(r.nodes, h)
	}
	// Rebuild sorted slice
	r.sorted = r.sorted[:0]
	for h := range r.nodes {
		r.sorted = append(r.sorted, h)
	}
	sort.Slice(r.sorted, func(i, j int) bool { return r.sorted[i] < r.sorted[j] })
}

// Lookup returns the node address responsible for the given key.
func (r *Ring) Lookup(key string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.sorted) == 0 {
		return "", fmt.Errorf("ring is empty")
	}
	h := hash(key)
	idx := sort.Search(len(r.sorted), func(i int) bool { return r.sorted[i] >= h })
	if idx == len(r.sorted) {
		idx = 0 // wrap around
	}
	return r.nodes[r.sorted[idx]], nil
}

// Replicas returns up to n distinct physical node addresses for the given key,
// starting at the primary and walking clockwise around the ring.
func (r *Ring) Replicas(key string, n int) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.sorted) == 0 {
		return nil, fmt.Errorf("ring is empty")
	}
	h := hash(key)
	idx := sort.Search(len(r.sorted), func(i int) bool { return r.sorted[i] >= h })
	if idx == len(r.sorted) {
		idx = 0
	}
	seen := make(map[string]bool)
	var out []string
	for i := 0; i < len(r.sorted) && len(out) < n; i++ {
		addr := r.nodes[r.sorted[(idx+i)%len(r.sorted)]]
		if !seen[addr] {
			seen[addr] = true
			out = append(out, addr)
		}
	}
	return out, nil
}

func hash(s string) uint32 {
	b := sha256.Sum256([]byte(s))
	return binary.BigEndian.Uint32(b[:4])
}
