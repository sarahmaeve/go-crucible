# Exercise 08: The Zombie Metric

**Application:** pipeline | **Difficulty:** Intermediate

## Symptoms

Multiple goroutines call `WindowedAggregator.Add` concurrently. Under the race detector the program reports a data race on the `samples` map. Without the race detector the map occasionally panics or silently drops data. The aggregator appears to work in single-threaded tests but fails under any real concurrent load.

## Reproduce

```bash
go test -race ./internal/transform/ -run TestExercise08 -v
```

## File to Investigate

`internal/transform/aggregate.go` — look at the `WindowedAggregator` struct and its `Add` method

Notice that `Add` reads and writes `a.samples` without holding any lock.

## What You Will Learn

- Go maps are not safe for concurrent use: simultaneous reads and writes cause undefined behaviour
- The race detector (`-race` flag) reliably catches these bugs at test time
- The two standard fixes: add a `sync.Mutex` to the struct and lock/unlock around map access, or replace the map with a `sync.Map`
- Why `-race` should be part of your regular test suite, not an occasional check

## Fixing It

Apply your fix, then run:

```bash
go test -race ./internal/transform/ -run TestExercise08 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
