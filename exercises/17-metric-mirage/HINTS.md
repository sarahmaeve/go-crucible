# Hints for Exercise 17: The Metric Mirage

## Hint 1: Direction

The function returns a map that has the right keys but the wrong label values — the rename rules had no effect. The renaming logic looks correct when you read it. The problem is not in the renaming itself, but in what happens to the result of the renaming.

## Hint 2: Narrower

Open `internal/transform/relabel.go`. Inside the loop in `Relabel`, the variable `m` is the loop variable — it holds a *copy* of the map value (because `types.Metric` is a struct). After `m.Labels = newLabels` is executed, `m` is updated — but the map at `result[key]` still holds the old copy. The modified `m` is discarded at the end of the iteration.

## Hint 3: Almost There

After applying the label rename, write the modified copy back to the result map:

```go
m.Labels = newLabels
result[key] = m  // write the updated copy back — this line is missing
```

Replace `_ = key` with `result[key] = m`. The loop variable `key` is there for exactly this purpose; the original code intentionally omits the write-back as the bug.
