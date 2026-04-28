// Leader election: only one coordinator node runs drift detection at a time.
// Uses a Redis key as a distributed lock (simplest approach before etcd).
// See docs/05-leader-election.md for a full explanation.
package coordinator

import (
	"context"
	"fmt"
	"os"
	"time"
)

const (
	leaderKey   = "company-brain:leader"
	leaseTTL    = 10 * time.Second
	renewInterval = 5 * time.Second
)

type Elector struct {
	nodeID string
	// rdb *redis.Client  -- add after go get github.com/redis/go-redis/v9
}

func NewElector(redisAddr string) *Elector {
	hostname, _ := os.Hostname()
	return &Elector{nodeID: hostname}
}

// Run campaigns for leadership and holds the lease until ctx is cancelled.
// Calls onLeader when this node wins, onFollower when it loses the lease.
func (e *Elector) Run(ctx context.Context, onLeader, onFollower func()) {
	fmt.Printf("[election] node %s starting campaign\n", e.nodeID)
	ticker := time.NewTicker(renewInterval)
	defer ticker.Stop()

	isLeader := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			won := e.tryAcquire(ctx)
			if won && !isLeader {
				isLeader = true
				fmt.Printf("[election] %s became leader\n", e.nodeID)
				go onLeader()
			} else if !won && isLeader {
				isLeader = false
				fmt.Printf("[election] %s lost leadership\n", e.nodeID)
				go onFollower()
			}
		}
	}
}

// tryAcquire attempts a SET NX PX on the leader key.
func (e *Elector) tryAcquire(_ context.Context) bool {
	// TODO: result, _ := e.rdb.SetNX(ctx, leaderKey, e.nodeID, leaseTTL)
	// return result
	return true // stub: always leader in single-node mode
}
