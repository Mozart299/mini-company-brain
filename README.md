# Company Brain

A distributed system that acts as a company's operational memory — ingesting data from GitHub, Slack, and Linear, storing it as structured knowledge, and detecting drift between what was planned and what's actually being built.

Built as both a learning project (distributed systems + OS concepts) and a prototype of the [YC RFS "AI Operating System for Companies"](https://www.ycombinator.com/rfs) idea.

---

## Architecture

```
[Data Sources]       [Ingestion Layer]       [Knowledge Store]      [Query / Alert Layer]
  GitHub API    -->  Ingestion Workers   -->  Distributed KV Store  -->  REST API
  Slack Feed    -->  (Go goroutines)     -->  (versioned, replicated)-->  Drift Detector
  Linear Tickets-->  Redis Streams       -->  Consistent Hash Ring   -->  Dashboard (soon)
```

### Components

| Component | Location | Description |
|---|---|---|
| Ingestion Workers | `ingestion/` | One goroutine per source (GitHub, Slack, Linear) |
| Message Queue | `queue/` | Redis Streams wrapper |
| Knowledge Store | `store/` | In-memory KV → BadgerDB (Milestone 2) |
| Coordinator | `coordinator/` | Leader election + drift detection |
| REST API | `api/` | Serves facts and drift alerts |
| Dashboard | `web/` | Next.js frontend (Milestone 4) |

---

## Tech Stack

| Layer | Choice |
|---|---|
| Ingestion / backend | Go |
| Message queue | Redis Streams |
| Knowledge store | In-memory → BadgerDB → etcd |
| API | Go `net/http` |
| Dashboard | Next.js |
| Local infra | Docker Compose |

---

## Getting Started

### Prerequisites

- Go 1.22+
- Docker

### Run locally

```bash
cp .env.example .env   # fill in your API keys
make up                # builds and starts everything
```

That's it. Docker Compose brings up Redis, 3 store nodes, ingestion workers, coordinator, API, and the dashboard in one command.

| Service     | URL                    |
|-------------|------------------------|
| Dashboard   | http://localhost:3000  |
| API         | http://localhost:8080  |

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
make logs s=api          # tail api logs
make logs s=coordinator  # tail coordinator logs
make logs s=ingestion    # tail ingestion logs
```

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `REDIS_ADDR` | `localhost:6379` | Redis address |
| `GITHUB_TOKEN` | — | GitHub personal access token |
| `GITHUB_REPO` | `owner/repo` | Repo to ingest (e.g. `myorg/myrepo`) |
| `LINEAR_API_KEY` | — | Linear API key |
| `LINEAR_TEAM_ID` | — | Linear team ID |
| `PORT` | `8080` | API server port |

---

## Milestones

### ✅ Milestone 1 — Single-node ingestion pipeline
- Three ingestion workers running as concurrent goroutines (GitHub, Slack, Linear)
- Redis Streams as the message queue with consumer groups
- In-memory KV store populated by a stream consumer
- REST API serving stored facts and drift alerts
- **Concepts covered:** goroutines, channels, context cancellation, Redis Streams, at-least-once delivery

### ✅ Milestone 2 — Distributed store
- BadgerDB replaces in-memory store — data now persists to disk per node
- Each store node runs as an independent HTTP server (`cmd/store`)
- Consistent hashing ring partitions keys across 3 nodes; virtual nodes (150×) ensure even distribution
- ReplicatedStore fans writes to N=2 nodes and falls back to replicas on read failure
- List queries all nodes and merges by highest version (distributed scatter-gather)
- `api` process auto-selects distributed mode when `STORE_NODES` env var is set
- **Concepts covered:** consistent hashing, replication, eventual consistency, fault tolerance

### ✅ Milestone 3 — Coordinator + Drift Detection
- Real Redis `SET NX` leader election with lease renewal and fence tokens
- Only the elected leader runs drift detection — followers pause automatically
- Two drift patterns: untracked commits (no ticket reference) and stale in-progress tickets (7+ days, no commits)
- Alerts written as `drift.alert:{id}` facts into the store — persistent across restarts
- API reads alerts directly from the store (no direct coordinator coupling)
- Coordinator connects to the same distributed store as the API
- **Concepts covered:** leader election, lease renewal, split-brain prevention, fencing tokens, distributed joins

### ✅ Milestone 4 — Query API + Dashboard
- `POST /query` endpoint sends stored facts as context to Claude (claude-sonnet-4-6) and returns a natural language answer
- Prompt caching on the system prompt reduces token cost on repeated queries
- CORS middleware added so Next.js dev server can call the API directly
- Next.js 15 dashboard with three panels: ingestion feed, drift alerts, query interface
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
│   ├── ingestion/      ← entry point: runs all ingestion workers
│   ├── api/            ← entry point: runs store consumer + HTTP server
│   └── coordinator/    ← entry point: runs leader election + drift detector
├── ingestion/
│   ├── github/         ← GitHub ingestion worker
│   ├── slack/          ← Slack mock feed worker
│   └── linear/         ← Linear ingestion worker
├── queue/              ← Redis Streams client
├── store/              ← KV store interface, partitioning, replication, consumer
├── coordinator/        ← Leader election, drift detection
├── api/                ← HTTP server and route handlers
├── pkg/types/          ← Shared event and fact types
├── docs/               ← Learning notes for each major concept
y└── docker-compose.yml  ← Multi-node local setup (Milestone 2+)
```


