# Hints for Exercise 18: The Ticking Leak

## Hint 1: Direction

Memory grows over time proportional to the number of loop iterations. No goroutines are leaking — it is heap objects. Something inside the loop allocates an object that is not freed promptly. Look at what the `select` statement creates on each iteration.

## Hint 2: Narrower

Open `internal/ingest/ticker.go` and look at the `Run` method. Inside the `select`, one of the cases is `case <-time.After(interval)`. `time.After` allocates a `*time.Timer` and registers it with the Go runtime. On the next iteration, a new `time.After` is called — but the previous timer's runtime registration is not cancelled. The old timer object stays alive until it fires naturally.

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

`ticker.Stop()` releases the ticker when `Run` returns, and only a single timer object is ever live at once.
