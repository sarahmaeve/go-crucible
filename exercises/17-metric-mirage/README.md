# Exercise 17: The Metric Mirage

**Application:** pipeline | **Difficulty:** Advanced

## Symptoms

`MetricRelabeler.Relabel` is called with a map of metrics and a set of rename rules. It returns a new map. The returned map contains the correct metric names, but none of the label keys have been renamed — the relabeling rules had no effect. No error is returned; the output looks structurally correct but the label data is unchanged.

## Reproduce

```bash
go test ./internal/transform/ -run TestExercise17 -v
```

## File to Investigate

`internal/transform/relabel.go` — look at the `Relabel` method on `MetricRelabeler`

Follow what happens to `m` (the loop variable) after the label renaming is applied. Is the modified value ever written back to `result`?

## What You Will Learn

- Map values in Go are not addressable — when you range over a map, the value is a copy
- Modifying the copy (`m.Labels = newLabels`) has no effect on the value stored in the map
- The fix: after modifying `m`, write it back with `result[key] = m`
- This is distinct from the map-reference bug in exercise 07 — here the outer container is correct; the problem is that the inner struct value is never reassigned

## Fixing It

Apply your fix, then run:

```bash
go test ./internal/transform/ -run TestExercise17 -v
```

See [HINTS.md](./HINTS.md) for progressive hints if you get stuck.
