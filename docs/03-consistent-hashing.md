# Consistent Hashing

## The Problem This Solves

We have 3 store nodes. When a fact arrives, which node stores it? The naive approach — `node = hash(key) % numNodes` — breaks badly when you add or remove a node: almost every key maps to a different node, causing a massive reshuffling of data.

**Consistent hashing** ensures that when a node is added or removed, only ~1/N of keys need to move.

## The Hash Ring

Imagine a circle (ring) with positions 0 to 2³²-1. We hash each node's address to place it on the ring. To find which node owns a key, we hash the key and walk clockwise until we hit a node.

```
         0
    ┌────────────┐
    │            │
    │   Store-1  │  ← hash("store-1") = 800M
    │            │
2.8B┤            ├ 1.2B
    │   Store-3  │  ← hash("store-3") = 1.2B
    │            │
    │   Store-2  │  ← hash("store-2") = 2.2B
    └────────────┘
        2.2B / 2.8B
```

Key "github.commit:abc" hashes to 900M → walks clockwise → hits Store-3 at 1.2B.

## Virtual Nodes

With only 3 physical nodes on the ring, the distribution is uneven — one node might own 60% of the keyspace. We fix this with **virtual nodes**: each physical node gets 150 spots on the ring (with different hashes).

```go
for i := 0; i < virtualNodes; i++ {
    h := hash(fmt.Sprintf("%s#%d", addr, i))
    ring[h] = addr
}
```

Now the ring has 450 points (3 × 150), and the load distributes evenly.

## Adding a Node (Why Only 1/N Keys Move)

Before adding Store-4: a key at position 500M maps to Store-1 (next clockwise node).  
After adding Store-4 at position 400M: only keys between 400M and Store-1's first virtual node move to Store-4. Everything else is unaffected.

In a `% N` scheme, adding node 4 would remap ~75% of all keys. With consistent hashing, only ~25% move.

## How We Use It

In `store/partition.go`, the `Ring` struct implements this. The `ReplicatedStore` in `store/replication.go` calls `ring.Lookup(key)` to find the primary node, then writes to N replicas.

```go
ring := store.NewRing()
ring.AddNode("store-1:7001")
ring.AddNode("store-2:7002")
ring.AddNode("store-3:7003")

node, _ := ring.Lookup("github.commit:abc123") // returns e.g. "store-2:7002"
```

## OS Analogy

Consistent hashing is like virtual memory page tables — a level of indirection that lets you remap physical resources without disrupting everything that depends on them.
