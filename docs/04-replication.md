# Replication

## The Problem This Solves

If we store each fact on exactly one node and that node crashes, the data is gone. Replication keeps copies of each fact on N nodes so the system survives failures.

## Replication Factor

We use N=2: every fact is written to 2 nodes. With 3 nodes total, you can lose any 1 node and still read every fact.

```
Write "github.commit:abc" → Store-2 (primary) + Store-3 (replica)
Write "linear.ticket:ENG-42" → Store-1 (primary) + Store-2 (replica)
```

The primary is the node the consistent hash ring returns. The replica is the **next clockwise node** on the ring.

## Write Path

```
Client
  │
  ▼
ReplicatedStore.Set(fact)
  ├── ring.Lookup(key) → "store-2:7002" (primary)
  ├── store2.Set(fact) ✓
  └── store3.Set(fact) ✓  ← next node clockwise
```

We write to both synchronously before returning success. If one fails, we still succeed (partial write is acceptable at replication factor 2 — the other replica holds the data).

## Read Path

```
ReplicatedStore.Get(key)
  ├── try primary (store-2) → success → return
  └── if primary down: try replica (store-3) → return
```

Reads always try the primary first. Falling back to a replica is transparent to the caller.

## Consistency Trade-off: Eventual Consistency

Our writes are **asynchronous** in Milestone 2 — the primary returns success after writing locally, then replicates in the background. This means:

- A read immediately after a write might return stale data from the replica
- The system is **eventually consistent**: replicas catch up within milliseconds

This is the same trade-off Amazon DynamoDB makes. It's acceptable here because our facts are append-only (new versions don't overwrite old ones — they have a `Version` timestamp).

## Quorum Reads/Writes (Future)

For stronger consistency, you can use quorum:
- **Write quorum**: write succeeds only after ⌈N/2⌉+1 nodes acknowledge
- **Read quorum**: read from ⌈N/2⌉+1 nodes and return the newest version

With N=3, quorum = 2. This guarantees reads always see the latest write (even if one node is behind), at the cost of higher latency.

## Handling Node Failures

When `ring.RemoveNode(addr)` is called (e.g., health check detects a node is down), the ring rebalances. The formerly unreachable node's keys now route to the next clockwise node, which may already have a replica.

**Anti-entropy**: a background process compares key sets between replicas and syncs missing entries. This is how Cassandra's "hinted handoff" works. We'll add this in Milestone 2.

## OS Analogy

Replication is like RAID-1 (disk mirroring). The same data lives on two disks. If one fails, the other takes over. The OS storage layer handles the routing transparently.
