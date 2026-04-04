# Hints for Exercise 05: The Nil Check That Lies

## Hint 1: Direction

The caller's `if client != nil` guard passes, yet the subsequent method call panics. That means `client` is not nil in the sense Go's `==` operator understands, but the underlying pointer is nil. This is a property of how interface values work internally.

## Hint 2: Narrower

Open `internal/client/client.go` and look at the error paths in `NewAuditClient`. What is returned when config loading fails? It is `(*KubeClient)(nil), nil` — a typed nil pointer stored in the `AuditClient` interface. An interface is only `== nil` when both its type and value parts are nil. Here the type part is `*KubeClient`, so the interface is not nil.

## Hint 3: Almost There

Replace every `return (*KubeClient)(nil), nil` in the error paths with `return nil, err`. Plain `nil` assigned to an interface variable sets both the type and value parts to nil, so the caller's `if client != nil` check will correctly return `false` and the panic is avoided.
