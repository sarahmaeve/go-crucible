# Exercise 14: The Forever Forwarder

**Application:** pipeline | **Difficulty:** Advanced

## Symptoms

`ForwardMetrics` is called with a channel that is eventually closed by the producer. After the channel is closed, the function should return. Instead it spins at 100% CPU forever. The test times out waiting for `ForwardMetrics` to exit. No goroutine leak detector is needed — the CPU spike makes it obvious.

## Reproduce

```bash
go test ./internal/ingest/ -run TestExercise14 -v
```

## File to Investigate

`internal/ingest/reader.go` — look at the `ForwardMetrics` function

Study the `select` statement inside the `for` loop. When `in` is closed, what does the `case m, ok := <-in:` branch return for `ok`? What does the code do with that information?

## What You Will Learn

- Reading from a closed channel in Go returns immediately with the channel's zero value and `ok == false`
- If the code ignores `ok` and `continue`s, the loop spins indefinitely — every iteration the closed channel case fires instantly
- The fix: when `!ok`, return from the function (the channel is exhausted)
- Always check the `ok` boolean when ranging over or selecting from channels that can be closed

## Fixing It

Apply your fix, then run:

```bash
go test ./internal/ingest/ -run TestExercise14 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
