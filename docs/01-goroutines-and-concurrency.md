# Goroutines and Concurrency

## The Problem This Solves

We need to pull data from GitHub, Slack, and Linear **at the same time** without each source blocking the others. If we did them sequentially, a slow GitHub API call would delay the Slack ingestion.

## What a Goroutine Is

A goroutine is a lightweight thread managed by the Go runtime — not the OS. You can run thousands of them cheaply. You launch one with `go`:

```go
go worker.Run(ctx) // starts immediately, doesn't block
```

The Go scheduler multiplexes goroutines onto a smaller pool of OS threads (M:N threading). When a goroutine blocks on I/O (network call, file read), the scheduler runs another goroutine on the same thread instead of wasting the CPU.

## How We Use It

In `cmd/ingestion/main.go`, we launch three goroutines — one per data source:

```go
go func() { errCh <- githubWorker.Run(ctx) }()
go func() { errCh <- slackWorker.Run(ctx) }()
go func() { errCh <- linearWorker.Run(ctx) }()
```

Each worker has its own poll loop with a `time.Ticker`. They run independently and publish events to the shared queue.

## Context and Cancellation

`context.Context` is how you signal a goroutine to stop. When the user presses Ctrl+C, `signal.NotifyContext` cancels the context, which propagates to every `<-ctx.Done()` select case in the workers.

```go
ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
defer cancel()
```

Every long-running function should accept a `ctx context.Context` as its first argument and respect cancellation.

## Channels for Communication

Goroutines communicate through channels, not shared memory. We use a buffered error channel to collect failures:

```go
errCh := make(chan error, 3) // buffer of 3 so goroutines don't block on send
```

## Common Pitfall: Goroutine Leaks

A goroutine leak happens when you start a goroutine that never exits. Always make sure every goroutine has a path to return — usually by selecting on `ctx.Done()`.

```go
for {
    select {
    case <-ctx.Done():
        return nil // clean exit
    case <-ticker.C:
        // do work
    }
}
```

## OS Analogy

Goroutines are like processes in an OS, but much cheaper. The Go scheduler is like the OS scheduler — it decides which goroutine runs on which CPU core. Unlike OS threads (~1MB stack each), goroutines start with ~8KB and grow only if needed.
