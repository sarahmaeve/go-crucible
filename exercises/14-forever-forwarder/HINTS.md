# Hints for Exercise 14: The Forever Forwarder

## Hint 1: Direction

After the input channel is closed, `ForwardMetrics` should return. Instead it loops forever. Reading from a closed channel in Go does not block — it returns immediately. If the loop keeps going, there must be something in the loop body that does not detect the closed channel and exit.

## Hint 2: Narrower

Open `internal/ingest/reader.go` and look at `ForwardMetrics`. The `select` case is `case m, ok := <-in:`. The `ok` variable is captured — but look at what the code does when `!ok`. It hits `continue`, which starts the next loop iteration, which immediately reads from the (still-closed) channel again, repeating forever.

## Hint 3: Almost There

When `ok` is `false`, the channel is closed and there is nothing more to read. Return from the function:

```go
case m, ok := <-in:
    if !ok {
        return nil  // channel closed; we are done
    }
    out <- m
```

Remove the `_ = m` and `continue` lines and replace them with this check.
