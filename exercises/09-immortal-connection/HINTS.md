# Hints for Exercise 09: The Immortal Connection

## Hint 1: Direction

The test tracks how many readers are open at once as well as how many are
eventually closed. It expects one close per opened reader and a peak of one live
reader. Something is opening a resource and never closing it.

## Hint 2: Narrower

Open `internal/audit/secrets.go` and find the
`reader := newSecretReader(...)` line inside the loop in
`AuditSecretExpiry`. There is no matching `Close`. A `defer` placed directly in
that loop is not enough: it would retain every reader until
`AuditSecretExpiry` returns.

## Hint 3: Almost There

Extract the open/read/close lifetime into a helper so its defer runs once per
iteration:

```go
func readSecretExpiry(data []byte, expiryStr string) (time.Time, error) {
    reader := newSecretReader(data)
    defer reader.Close()
    return parseExpiryFromReader(reader, expiryStr)
}
```

Call the helper from the loop. Each reader then closes before the next
iteration begins.
