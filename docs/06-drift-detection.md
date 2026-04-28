# Drift Detection

## The Problem This Solves

Engineering teams frequently build things not in any ticket, or let tickets go stale while work diverges. "Drift" is the gap between **what was planned** (tickets, specs) and **what was built** (commits, PRs).

The coordinator's job is to surface this drift automatically, without anyone having to manually cross-reference Linear and GitHub.

## What We Compare

| Source | "Planned" signals | "Actual" signals |
|---|---|---|
| Linear | Tickets in `todo` / `in_progress` | Tickets moved to `done` |
| GitHub | — | Commits, merged PRs |

**Drift scenario examples:**
1. Commit message has no matching ticket ID → untracked work
2. PR merged into `auth/` module but no auth ticket exists → feature scope creep
3. Ticket marked `in_progress` for 14+ days but no commits reference it → stalled ticket
4. Multiple PRs merged to `payments/` but the payments epic is still `backlog` → misaligned planning

## Our Algorithm (v1)

Simple heuristic in `coordinator/drift.go`:

```
1. Load all commits from store (prefix "github.commit:")
2. Load all tickets from store (prefix "linear.ticket:")
3. For each commit:
     if commit.message contains no ticket ID → raise DriftAlert
4. For each in_progress ticket older than 14 days:
     if no commits reference ticket.ID → raise DriftAlert
```

This is intentionally naive and will have false positives (commits like "fix typo" that don't need a ticket). The LLM query endpoint (`POST /query`) can be used to ask smarter questions over the same data.

## State Comparison as a Distributed Problem

Drift detection is a **distributed join** — we're joining two datasets (commits from GitHub, tickets from Linear) that live on different store nodes. In Milestone 3:

1. The leader coordinator fetches all commits from partition ring
2. Fetches all tickets from partition ring
3. Joins them in memory and runs the heuristic
4. Writes resulting `DriftAlert` facts back to the store

This is similar to how a distributed SQL engine executes a JOIN across shards: gather, shuffle, join, write results.

## Alert Lifecycle

```
DriftAlert created → stored in KV with key "drift.alert:{id}"
                   → exposed via GET /alerts
                   → (future) sent to Slack / email
                   → resolved when commit/ticket gap closes
                   → stored with "resolved_at" timestamp
```

## LLM Queries (Milestone 4)

The `POST /query` endpoint lets you ask natural language questions over all stored facts:

```
POST /query
{"question": "Is there any engineering work that doesn't match our planned tickets?"}
```

All facts (commits, tickets, Slack messages, alerts) are sent as context to Claude. The heuristic detector catches obvious gaps; the LLM catches subtle ones — like a refactor that doesn't mention a ticket but clearly corresponds to a security initiative discussed in Slack.

## Why This Matters for the YC Idea

The YC RFS describes a "Company Brain" that helps leadership understand what's actually happening vs what they think is happening. Drift detection is the core signal — it's what makes this a product rather than just a data pipeline.
