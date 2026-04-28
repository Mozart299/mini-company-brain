# Project: Company Brain (Mini)

## What I'm Building

A distributed system that acts as a "Company Brain" — ingesting data from multiple sources (GitHub, Slack, ticketing systems), storing structured knowledge, and surfacing insights like drift detection (when what's being built diverges from what was planned).

This is both a **learning project** (distributed systems + OS concepts) and a **real product prototype** aligned with the YC Summer 2026 RFS idea "The AI Operating System for Companies" by Diana Hu.

---

## Learning Goals

This project is intentionally designed to touch core CS concepts:

**Distributed Systems**
- Message queues and event-driven ingestion (Kafka or Redis Streams)
- Consistent hashing / partitioning for knowledge storage
- Replication and fault tolerance across nodes
- Consensus / coordinator pattern for drift detection

**Operating Systems**
- Process/goroutine scheduling for concurrent ingestion workers
- Context switching between pipeline stages
- Memory management for large knowledge graph state
- Inter-process communication patterns

---

## Architecture Overview

```
[Data Sources]          [Ingestion Layer]        [Knowledge Store]       [Query / Alert Layer]
  GitHub API      -->   Ingestion Workers   -->   Distributed KV Store  -->  REST / MCP API
  Slack Feed      -->   (Go goroutines)     -->   (versioned, replicated)-->  Drift Detector
  Linear/Tickets  -->   Message Queue       -->   Knowledge Graph         -->  AI Query Interface
                        (Kafka/Redis)
```

### Components

**1. Ingestion Workers** (`/ingestion`)
- Go goroutines, one per data source
- Pull from GitHub (commits, PRs, issues), Slack (mock feed), Linear (tickets)
- Publish normalized events to a message queue
- Demonstrates: concurrency, IPC, at-least-once delivery

**2. Message Queue** (`/queue`)
- Redis Streams or a minimal Kafka setup
- Buffers events between ingestion and storage
- Demonstrates: distributed messaging, backpressure, consumer groups

**3. Knowledge Store** (`/store`)
- Distributed key-value store built from scratch (or using etcd/Redis as backing)
- Stores structured "facts": decisions, PRs, ticket states, meeting notes
- Versioned — every fact has a timestamp and source
- Demonstrates: replication, consistency, partitioning

**4. Coordinator / Drift Detector** (`/coordinator`)
- Runs as a separate process
- Compares "planned work" (tickets/specs) against "actual work" (commits/PRs)
- Flags divergence — e.g. "3 PRs merged into auth module but no ticket exists for this"
- Demonstrates: leader election, consensus, distributed state comparison

**5. Query API** (`/api`)
- REST + optional MCP server interface
- Answers questions like: "What decisions were made about X last sprint?"
- Powers a simple Next.js dashboard
- Demonstrates: serving distributed state, caching

**6. Dashboard** (`/web`)
- Next.js frontend
- Shows: ingestion feed, knowledge graph state, active drift alerts
- Simple — UI is not the focus

---

## Tech Stack

| Layer | Choice | Why |
|---|---|---|
| Ingestion workers | Go | Goroutines make concurrent workers ergonomic |
| Message queue | Redis Streams | Lightweight, easy local setup; swap for Kafka later |
| Knowledge store | Go + BadgerDB or etcd | Embeddable KV with good Go support |
| Coordinator | Go | Same process pool, shares types |
| API | Go (Fiber or net/http) | Consistent with rest of backend |
| Dashboard | Next.js | Peter's primary frontend stack |
| Infra (local) | Docker Compose | Multi-node simulation locally |

---

## Repo Structure

```
company-brain/
├── CLAUDE.md               ← you are here
├── docker-compose.yml      ← spins up all nodes locally
├── ingestion/
│   ├── github/             ← GitHub ingestion worker
│   ├── slack/              ← Slack mock feed worker
│   └── linear/             ← Linear/ticket ingestion worker
├── queue/
│   └── streams.go          ← Redis Streams wrapper
├── store/
│   ├── kv.go               ← Core KV store interface
│   ├── replication.go      ← Replication logic
│   └── partition.go        ← Consistent hashing
├── coordinator/
│   ├── election.go         ← Leader election
│   └── drift.go            ← Drift detection logic
├── api/
│   └── server.go           ← REST API
└── web/                    ← Next.js dashboard
```

---

## Build Order (Milestones)

### Milestone 1 — Single-node ingestion pipeline (Week 1–2)
- GitHub ingestion worker pulling real commits via API
- Redis Streams as the queue
- Simple in-memory KV store (no replication yet)
- Goal: events flowing end-to-end, stored and queryable

### Milestone 2 — Distributed store (Week 3–4)
- Replace in-memory store with BadgerDB
- Add consistent hashing to partition keys across 3 simulated nodes (Docker)
- Add replication: each key written to N=2 nodes
- Goal: data survives a node going down

### Milestone 3 — Coordinator + Drift Detection (Week 5–6)
- Leader election (simple: etcd lease or manual heartbeat)
- Drift detector compares ticket state vs commit activity
- Alerts surface via API
- Goal: system flags "engineering building something not in tickets"

### Milestone 4 — Query API + Dashboard (Week 7–8)
- REST API over the knowledge store
- Natural language queries via an LLM call (Anthropic API)
- Next.js dashboard showing ingestion feed + drift alerts
- Goal: demo-able product

---

## Key Design Constraints

- **Fault tolerant**: system continues if one ingestion worker or store node dies
- **Eventually consistent**: writes propagate asynchronously; reads may be slightly stale
- **Observable**: every component emits structured logs; coordinator tracks node health
- **Modular**: each component runs as an independent process, communicates over network

---

## Context

Builder: Peter Ssendegeya — Frontend/Flutter engineer at AirQo, Kampala. Strong in Next.js, Flutter, Firebase. Learning Go and distributed systems fundamentals. This project is designed to be both a portfolio piece and a genuine prototype of a startup idea.

Reference: YC RFS Summer 2026 — "The AI Operating System for Companies" (Diana Hu) and "Company Brain" (Tom Blomfield).

Prior conversation context: explored OS + distributed systems learning projects including custom shell, user-space thread library, Raft consensus, distributed KV store, and combined cluster resource manager.
