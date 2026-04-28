# Redis Streams

## The Problem This Solves

Ingestion workers produce events at unpredictable rates. The knowledge store might be temporarily slow or down. We need a buffer between producers and consumers so events aren't lost and neither side needs to know about the other's pace — this is **backpressure**.

## What Redis Streams Are

A Redis Stream is an append-only log, like a simplified Kafka topic. Each entry has:
- An auto-generated ID: `timestamp-sequenceNumber` (e.g. `1714300000000-0`)
- A map of field-value pairs (we store our serialized event as a `data` field)

```
XADD company-brain:events * data '{"id":"gh-123","kind":"github.commit",...}'
```

The `*` tells Redis to generate the ID. Entries stay in the stream forever (or until you trim with `MAXLEN`).

## Consumer Groups

A consumer group lets multiple consumers share the work of processing a stream. Each message is delivered to **one** consumer in the group (load balancing), and consumers must acknowledge (`XACK`) messages after processing.

```
XGROUP CREATE company-brain:events store-workers $ MKSTREAM
XREADGROUP GROUP store-workers worker-1 COUNT 10 STREAMS company-brain:events >
```

The `>` means "give me only new messages not yet delivered to this group."

## How We Use It

```
[GitHub Worker] ──┐
[Slack Worker]  ──┼──► company-brain:events (Redis Stream) ──► [Store Consumer Group]
[Linear Worker] ──┘                                          └──► [Coordinator Consumer Group]
```

Two consumer groups can read the same stream independently — the store workers and coordinator each get every message.

## Delivery Guarantees

Redis Streams give **at-least-once delivery**: if a consumer crashes before acking, the message stays in the pending list and will be redelivered. Your consumer logic must be idempotent (safe to run twice).

We use the event `ID` field as a deduplication key in the store.

## Compared to Kafka

| Feature | Redis Streams | Kafka |
|---|---|---|
| Setup | Single Redis instance | ZooKeeper + brokers |
| Throughput | ~100K msgs/sec | ~1M msgs/sec |
| Retention | In-memory (configurable) | Disk-backed, days/weeks |
| Use case | Low-medium volume, simple setup | High volume, long retention |

Redis Streams are the right choice for Milestone 1. The queue interface in `queue/streams.go` is designed so you can swap in Kafka later without changing the ingestion workers.

## Key Redis Commands

```bash
XADD stream * field value        # publish
XREAD COUNT 10 STREAMS stream 0  # read (no consumer group)
XGROUP CREATE stream group $ MKSTREAM
XREADGROUP GROUP g consumer COUNT 10 STREAMS stream >
XACK stream group message-id
XLEN stream                      # how many messages
```
