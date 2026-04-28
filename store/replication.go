// ReplicatedStore fans reads and writes across multiple store nodes using the
// consistent hash ring. See docs/04-replication.md.
package store

import (
	"context"
	"fmt"

	"company-brain/pkg/types"
)

const replicationFactor = 2 // write to this many distinct nodes per key

// ReplicatedStore implements Store by routing operations to the correct nodes.
type ReplicatedStore struct {
	ring    *Ring
	clients map[string]Store // addr -> NodeClient for that node
}

func NewReplicatedStore(ring *Ring) *ReplicatedStore {
	return &ReplicatedStore{ring: ring, clients: make(map[string]Store)}
}

// RegisterNode adds a store node to the ring and records its client.
func (rs *ReplicatedStore) RegisterNode(addr string, s Store) {
	rs.ring.AddNode(addr)
	rs.clients[addr] = s
}

// Set writes the fact to replicationFactor nodes: primary + clockwise successors.
func (rs *ReplicatedStore) Set(ctx context.Context, fact types.Fact) error {
	targets, err := rs.ring.Replicas(fact.Key, replicationFactor)
	if err != nil {
		return err
	}
	var errs []error
	for _, addr := range targets {
		if s, ok := rs.clients[addr]; ok {
			if err := s.Set(ctx, fact); err != nil {
				errs = append(errs, fmt.Errorf("node %s: %w", addr, err))
				fmt.Printf("[replicated] write to %s failed: %v\n", addr, err)
			}
		}
	}
	if len(errs) == len(targets) {
		return fmt.Errorf("all replicas failed: %v", errs)
	}
	return nil
}

// Get reads from the primary; falls back to each replica on failure.
func (rs *ReplicatedStore) Get(ctx context.Context, key string) (types.Fact, error) {
	targets, err := rs.ring.Replicas(key, replicationFactor)
	if err != nil {
		return types.Fact{}, err
	}
	for _, addr := range targets {
		if s, ok := rs.clients[addr]; ok {
			if f, err := s.Get(ctx, key); err == nil {
				return f, nil
			}
		}
	}
	return types.Fact{}, fmt.Errorf("key not found on any replica: %s", key)
}

// Delete removes the key from all replicas.
func (rs *ReplicatedStore) Delete(ctx context.Context, key string) error {
	targets, err := rs.ring.Replicas(key, replicationFactor)
	if err != nil {
		return err
	}
	for _, addr := range targets {
		if s, ok := rs.clients[addr]; ok {
			s.Delete(ctx, key)
		}
	}
	return nil
}

// List queries every node and merges results, keeping the highest-version fact
// per key. This is necessary because different keys hash to different primaries.
func (rs *ReplicatedStore) List(ctx context.Context, prefix string) ([]types.Fact, error) {
	seen := make(map[string]types.Fact)
	for _, s := range rs.clients {
		facts, err := s.List(ctx, prefix)
		if err != nil {
			continue // tolerate a node being down
		}
		for _, f := range facts {
			if existing, ok := seen[f.Key]; !ok || f.Version > existing.Version {
				seen[f.Key] = f
			}
		}
	}
	out := make([]types.Fact, 0, len(seen))
	for _, f := range seen {
		out = append(out, f)
	}
	return out, nil
}
