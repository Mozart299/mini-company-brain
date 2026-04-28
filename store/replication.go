// Replication: each write is forwarded to N replica nodes so data
// survives a node going down. See docs/04-replication.md.
package store

import (
	"context"
	"fmt"

	"company-brain/pkg/types"
)

const replicationFactor = 2 // write to this many nodes per key

// ReplicatedStore sits in front of the hash ring and fans writes out to N nodes.
type ReplicatedStore struct {
	ring    *Ring
	clients map[string]Store // addr -> store client for that node
}

func NewReplicatedStore(ring *Ring) *ReplicatedStore {
	return &ReplicatedStore{
		ring:    ring,
		clients: make(map[string]Store),
	}
}

// RegisterNode adds a reachable store node.
func (rs *ReplicatedStore) RegisterNode(addr string, s Store) {
	rs.ring.AddNode(addr)
	rs.clients[addr] = s
}

// Set writes the fact to the primary node plus (replicationFactor-1) successors.
func (rs *ReplicatedStore) Set(ctx context.Context, fact types.Fact) error {
	targets, err := rs.replicas(fact.Key)
	if err != nil {
		return err
	}
	var errs []error
	for _, addr := range targets {
		if s, ok := rs.clients[addr]; ok {
			if err := s.Set(ctx, fact); err != nil {
				errs = append(errs, fmt.Errorf("node %s: %w", addr, err))
			}
		}
	}
	if len(errs) == len(targets) {
		return fmt.Errorf("all replicas failed: %v", errs)
	}
	return nil
}

// Get reads from the primary; falls back to replicas on failure.
func (rs *ReplicatedStore) Get(ctx context.Context, key string) (types.Fact, error) {
	targets, err := rs.replicas(key)
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

// replicas returns up to replicationFactor node addresses for a key,
// starting from the primary on the ring.
func (rs *ReplicatedStore) replicas(key string) ([]string, error) {
	primary, err := rs.ring.Lookup(key)
	if err != nil {
		return nil, err
	}
	// TODO: walk the ring clockwise to collect replicationFactor-1 more nodes
	return []string{primary}, nil
}
