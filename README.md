# Company Brain

A distributed system that acts as a company's operational memory — ingesting data from GitHub, Slack, and Linear, storing it as structured knowledge, and detecting drift between what was planned and what's actually being built.

Built as both a learning project (distributed systems + OS concepts) and a prototype of the [YC RFS "AI Operating System for Companies"](https://www.ycombinator.com/rfs) idea.

---

## Architecture

```
[Data Sources]       [Ingestion Layer]       [Knowledge Store]      [Query / Alert Layer]
  GitHub API    -->  Ingestion Workers   -->  Distributed KV Store  -->  REST API
  Slack Feed    -->  (Go goroutines)     -->  (BadgerDB, replicated) -->  Drift Detector
  Linear Tickets-->  Redis Streams       -->  Consistent Hash Ring   -->  Next.js Dashboard
```

### Components

| Component | Location | Description |
|---|---|---|
| Ingestion Workers | `ingestion/` | One goroutine per source (GitHub, Slack, Linear) |
| Message Queue | `queue/` | Redis Streams wrapper with consumer groups |
| Knowledge Store | `store/` | BadgerDB-backed distributed KV with consistent hashing |
| Coordinator | `coordinator/` | Leader election + drift detection |
| REST API | `api/` | Serves facts, alerts, and natural language queries |
| Dashboard | `web/` | Next.js 15 frontend with shadcn/ui |

---

## Tech Stack

| Layer | Choice |
|---|---|
| Ingestion / backend | Go |
| Message queue | Redis Streams |
| Knowledge store | BadgerDB (embedded, per node) |
| API | Go `net/http` |
| Dashboard | Next.js 15 + shadcn/ui |
| Infrastructure | Docker Compose |

---

## Getting Started

### Prerequisites

- Docker

### Run

```bash
cp .env.example .env   # fill in your API keys
make up                # builds and starts everything
```

Docker Compose brings up all 8 services in one command: Redis, 3 store nodes, ingestion workers, coordinator, API, and the dashboard.

| Service   | URL                   |
|-----------|-----------------------|
| Dashboard | http://localhost:3000 |
| API       | http://localhost:8080 |

### Environment variables

Copy `.env.example` to `.env` and fill in:

| Variable | Description |
|---|---|
| `ANTHROPIC_API_KEY` | Required for the natural language query feature |
| `GITHUB_TOKEN` | GitHub personal access token for real commit ingestion |
| `GITHUB_REPO` | Repo to watch, e.g. `myorg/myrepo` |
| `LINEAR_API_KEY` | Linear API key for ticket ingestion |
| `LINEAR_TEAM_ID` | Linear team ID to filter tickets |

### Query the API directly

```bash
curl localhost:8080/health
curl localhost:8080/facts | jq
curl "localhost:8080/facts?prefix=github.commit" | jq
curl localhost:8080/alerts | jq
curl -X POST localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"question": "What commits were made recently?"}' | jq
```

### Stream logs

```bash
make logs s=api          # tail API logs
make logs s=coordinator  # tail coordinator logs
make logs s=ingestion    # tail ingestion logs
```

---

## Milestones

### ✅ Milestone 1 — Single-node ingestion pipeline
- Three ingestion workers running as concurrent goroutines (GitHub, Slack, Linear)
- Redis Streams as the message queue with consumer groups and at-least-once delivery
- In-memory KV store populated by a stream consumer
- REST API serving stored facts
- **Concepts covered:** goroutines, channels, context cancellation, Redis Streams

### ✅ Milestone 2 — Distributed store
- BadgerDB replaces in-memory store — data persists to disk per node
- Each store node runs as an independent HTTP server (`cmd/store`)
- Consistent hashing ring partitions keys across 3 nodes with 150 virtual nodes per physical node
- ReplicatedStore fans writes to N=2 nodes; reads fall back to replicas on failure
- `List` does scatter-gather across all nodes, merging by highest version
- **Concepts covered:** consistent hashing, replication, eventual consistency, fault tolerance

### ✅ Milestone 3 — Coordinator + drift detection
- Redis `SET NX` leader election with lease renewal and fence tokens
- Only the elected leader runs drift detection — followers pause automatically on leadership loss
- Two drift patterns: untracked commits (no ticket reference) and stale in-progress tickets (7+ days)
- Alerts stored as `drift.alert:{id}` facts in the KV store — persistent across restarts
- API reads alerts directly from the store; no direct coupling to the coordinator
- **Concepts covered:** leader election, lease renewal, split-brain prevention, fencing tokens

### ✅ Milestone 4 — Query API + dashboard
- `POST /query` loads all facts as context and queries Claude (claude-sonnet-4-6)
- Prompt caching on the system prompt reduces token cost on repeated queries
- Next.js 15 dashboard with shadcn/ui: ingestion feed, drift alerts panel, query interface
- All panels poll the API every 5 seconds for live updates
- **Concepts covered:** LLM-augmented retrieval, prompt caching, serving distributed state

---

## Learning Docs

Concept explanations written alongside the code — each one ties the theory directly to how it's used in this system.

| Doc | Concept |
|---|---|
| [`docs/01-goroutines-and-concurrency.md`](docs/01-goroutines-and-concurrency.md) | Goroutines, channels, context cancellation |
| [`docs/02-redis-streams.md`](docs/02-redis-streams.md) | Redis Streams, consumer groups, backpressure |
| [`docs/03-consistent-hashing.md`](docs/03-consistent-hashing.md) | Hash ring, virtual nodes, key remapping |
| [`docs/04-replication.md`](docs/04-replication.md) | Replication factor, read/write paths, eventual consistency |
| [`docs/05-leader-election.md`](docs/05-leader-election.md) | Leader election, leases, split-brain, fencing |
| [`docs/06-drift-detection.md`](docs/06-drift-detection.md) | Drift detection, distributed joins, alert lifecycle |

---

## Project Structure

```
company-brain/
├── cmd/
│   ├── ingestion/      ← runs all ingestion workers
│   ├── store/          ← store node HTTP server
│   ├── api/            ← store consumer + REST API
│   └── coordinator/    ← leader election + drift detector
├── ingestion/
│   ├── github/         ← GitHub ingestion worker
│   ├── slack/          ← Slack mock feed worker
│   └── linear/         ← Linear ingestion worker
├── queue/              ← Redis Streams client
├── store/              ← BadgerDB, consistent hashing, replication, node server/client
├── coordinator/        ← leader election, drift detection
├── api/                ← HTTP handlers + Anthropic query
├── pkg/types/          ← shared event and fact types
├── web/                ← Next.js dashboard
├── docs/               ← learning notes for each major concept
├── docker-compose.yml  ← full stack (all 8 services)
└── .env.example        ← environment variable template
```

---

Builder: Peter Ssendegeya
