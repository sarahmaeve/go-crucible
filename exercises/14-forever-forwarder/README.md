# Exercise 14: The Forever Forwarder

**Application:** pipeline | **Difficulty:** Advanced

## Symptoms

`ForwardMetrics` has two lifecycle failures. When the producer closes the input,
it spins at 100% CPU instead of returning. When the downstream consumer stops
receiving, it can block forever on the output send even after its context is
cancelled.

## Reproduce

```bash
go test ./internal/ingest/ -run TestExercise14 -v
```

## File to Investigate

`internal/ingest/reader.go` — look at the `ForwardMetrics` function

Study the `select` statement inside the `for` loop. When `in` is closed, what
does the `case m, ok := <-in:` branch return for `ok`? Then inspect the send to
`out`: can cancellation interrupt it?

## What You Will Learn

- Reading from a closed channel in Go returns immediately with the channel's zero value and `ok == false`
- If the code ignores `ok` and `continue`s, the loop spins indefinitely — every iteration the closed channel case fires instantly
- The fix: when `!ok`, return from the function (the channel is exhausted)
- A plain channel send is a blocking operation; wrap it in a `select` with
  `ctx.Done()` when cancellation must remain effective
- Check the `ok` boolean for explicit receives from channels that can be
  closed, including receives inside `select`; a `for range` loop handles
  closure automatically

## Fixing It

Apply your fix, then run:

```bash
go test ./internal/ingest/ -run TestExercise14 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
