# Hints for Exercise 18: The Ticking Allocation

## Hint 1: Direction

The allocation count grows in direct proportion to the number of polls. No
goroutine or live heap object is leaking. Look at what the `select` statement
constructs afresh on every iteration.

## Hint 2: Narrower

Open `internal/ingest/ticker.go` and look at the `Run` method. Inside the
`select`, `case <-time.After(interval)` allocates a new one-shot timer. This
loop waits for that timer to fire and then repeats, allocating another one.
A recurring ticker can provide every wake-up from one allocation.

## Hint 3: Almost There

Replace `time.After` with `time.NewTicker`, created once before the loop:

```go
ticker := time.NewTicker(interval)
defer ticker.Stop()

for {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case <-ticker.C:
        m, err := source.Read(ctx)
        if err != nil {
            return err
        }
        select {
        case out <- m:
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}
```

`ticker.Stop()` ends the recurring schedule when `Run` returns. More
importantly for this exercise, one ticker replaces the per-poll allocation.

This changes the schedule from “wait an interval after the previous iteration”
to a fixed ticker cadence. That matches `Run`'s “polls every interval” contract;
if the intended contract were delay-after-work instead, use one
`time.NewTimer` and reset it after each completed iteration.
