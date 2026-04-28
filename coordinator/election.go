// Leader election using Redis SET NX as a distributed lock.
// See docs/05-leader-election.md for a full explanation including fencing tokens.
package coordinator

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	leaderKey     = "company-brain:leader"
	leaseTTL      = 10 * time.Second
	renewInterval = 4 * time.Second // must be less than leaseTTL
)

type Elector struct {
	nodeID string
	rdb    *redis.Client
	token  int64 // monotonic fence token — increments each time this node wins
}

func NewElector(redisAddr string) *Elector {
	hostname, _ := os.Hostname()
	return &Elector{
		nodeID: hostname,
		rdb:    redis.NewClient(&redis.Options{Addr: redisAddr}),
	}
}

// Run campaigns for the leader slot every renewInterval.
// Calls onLeader (in a goroutine) when this node wins, onFollower when it loses.
func (e *Elector) Run(ctx context.Context, onLeader, onFollower func()) {
	fmt.Printf("[election] node %s starting campaign\n", e.nodeID)
	ticker := time.NewTicker(renewInterval)
	defer ticker.Stop()

	isLeader := false
	for {
		select {
		case <-ctx.Done():
			if isLeader {
				// Release the lease immediately so a peer can take over fast.
				e.rdb.Del(context.Background(), leaderKey)
			}
			return
		case <-ticker.C:
			if isLeader {
				if e.renew(ctx) {
					continue
				}
				isLeader = false
				fmt.Printf("[election] %s lost leadership\n", e.nodeID)
				go onFollower()
			} else {
				if e.tryAcquire(ctx) {
					isLeader = true
					fmt.Printf("[election] %s became leader (fence token=%d)\n", e.nodeID, e.token)
					go onLeader()
				}
			}
		}
	}
}

// tryAcquire attempts SET NX — succeeds only if no other node holds the key.
// Encodes nodeID + token so renew can verify ownership.
func (e *Elector) tryAcquire(ctx context.Context) bool {
	val := fmt.Sprintf("%s:%d", e.nodeID, e.token+1)
	ok, err := e.rdb.SetNX(ctx, leaderKey, val, leaseTTL).Result()
	if ok && err == nil {
		e.token++
		return true
	}
	return false
}

// renew extends the lease only if this node's token still matches the stored value.
// If the key was taken by another node (split-brain), renew returns false.
func (e *Elector) renew(ctx context.Context) bool {
	current, err := e.rdb.Get(ctx, leaderKey).Result()
	if err != nil {
		return false
	}
	if current != fmt.Sprintf("%s:%d", e.nodeID, e.token) {
		return false
	}
	return e.rdb.Expire(ctx, leaderKey, leaseTTL).Err() == nil
}

// Token returns the current fence token.
// Downstream systems should reject writes carrying a lower token than the last seen.
func (e *Elector) Token() int64 { return e.token }
