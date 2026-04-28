# Leader Election

## The Problem This Solves

We might run multiple coordinator processes for redundancy (if one crashes, another takes over). But drift detection should only run on **one** node at a time — otherwise we'd generate duplicate alerts. Leader election lets multiple nodes agree on which one is currently the leader.

## The Core Idea

All coordinator nodes race to claim a "leader" slot. Only one can hold it at a time. The holder must periodically renew its claim (heartbeat). If it fails to renew (because it crashed), another node wins.

## Our Approach: Redis SET NX PX

Redis supports atomic conditional writes. `SET key value NX PX ttl` sets the key **only if it doesn't exist**, with a TTL in milliseconds. If it succeeds, you're the leader. If it fails, someone else already holds the key.

```go
// Try to become leader
result, _ := rdb.SetNX(ctx, "company-brain:leader", nodeID, 10*time.Second)
if result {
    // I am the leader
}
```

The 10-second TTL is the **lease duration**. The leader must renew every ~5 seconds:

```go
// Renew: only update TTL if I'm still the holder
rdb.Expire(ctx, "company-brain:leader", 10*time.Second)
```

If the leader crashes, the key expires after 10 seconds and another node claims it.

## Timeline

```
t=0s:  Node-A wins SET NX → becomes leader
t=5s:  Node-A renews (EXPIRE resets TTL to 10s)
t=10s: Node-A crashes, stops renewing
t=20s: Key expires in Redis
t=21s: Node-B wins SET NX → becomes new leader
       (max gap = lease TTL = 10s)
```

## The Fencing Problem

What if Node-A is slow (not crashed, just paused by GC) and misses its renewal? Node-B becomes leader. Then Node-A wakes up and **also** thinks it's the leader. This is a **split-brain**.

Fix: attach a monotonic **fence token** (a counter) to each lease. Storage nodes reject writes with a lower fence token than the last seen one.

```
Node-A has token=1, Node-B (new leader) has token=2
Node-A tries to write with token=1 → rejected
Node-B writes with token=2 → accepted
```

We'll add fence tokens in Milestone 3.

## Alternatives

| Method | Complexity | Guarantees |
|---|---|---|
| Redis SET NX | Simple | Good (with fencing) |
| etcd lease | Medium | Strong (Raft-backed) |
| ZooKeeper ephemeral node | Complex | Strong |
| Raft (custom) | High | Strongest |

Redis is the right call for Milestone 1. etcd is a natural upgrade if we need stronger guarantees.

## How We Use It

`coordinator/election.go` implements the `Elector` struct. The `Run` method polls every 5 seconds:

```go
elector.Run(ctx,
    func() { go detector.Run(ctx) },     // called when we win
    func() { fmt.Println("stepped down") }, // called when we lose
)
```

The drift detector only runs when this node is the elected leader.
