# Hints for Exercise 08: The Zombie Metric

## Hint 1: Direction

Multiple goroutines are calling `Add` at the same time. The race detector flags a read/write conflict on the `samples` map. Maps in Go have no built-in concurrency safety. Something needs to prevent simultaneous access.

## Hint 2: Narrower

Open `internal/transform/aggregate.go`. The `WindowedAggregator` struct has a
`samples` field but no synchronization primitive. Both the map and the slice
read-modify-write performed by `append` belong to one invariant, so protect the
whole operation with a mutex.

## Hint 3: Almost There

Add a `mu sync.Mutex` field to `WindowedAggregator`, then lock/unlock around both the read and the write in `Add`:

```go
func (a *WindowedAggregator) Add(m types.Metric) {
    a.mu.Lock()
    defer a.mu.Unlock()
    a.samples[m.Name] = append(a.samples[m.Name], m.Value)
}
```

Also protect `Flush` in the same way if it can be called concurrently with `Add`.
