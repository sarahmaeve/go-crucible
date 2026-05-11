# Hints for Exercise 22: The Hollow Recovery

## Hint 1: Direction

Open `internal/worker/pool.go` and look at `Pool.processOne`. The
function uses `defer` to register a panic-recovery handler before
calling the user-supplied `ProcessFunc`. The handler exists and is
correctly structured — `recover()` is called, the panic value is
checked, an error is constructed.

So why does the panic still escape? Walk through what `defer` actually
defers. There are two functions involved in the recovery: an anonymous
function (`func() { ... }()`) registered via `defer`, and the
`recoverPanic` method it calls. Ask yourself: which of these two is
*the* deferred function?

## Hint 2: Narrower

The Go specification is precise about when `recover()` works:

> The return value of recover is nil if any of the following conditions
> holds:
>
>   - panic's argument was nil;
>   - the goroutine is not panicking;
>   - **recover was not called directly by a deferred function**.

The emphasis on "directly" is doing real work. `recover()` walks one
frame up to see whether *that* frame was registered via `defer`. In
this code, the frame above `recover()` is `recoverPanic`. Was
`recoverPanic` the function passed to `defer`?

Open `processOne` and answer that question literally. What's the
argument to `defer` on the line in question?

## Hint 3: Almost There

The fix is one line.

The current code is:

```go
defer func() {
    p.recoverPanic(&r)
}()
```

The `defer` registers an anonymous function. When that anonymous
function runs during panic unwinding, it calls `recoverPanic`.
`recoverPanic` then calls `recover()` — but `recover()` is now two
frames inside the deferred call, not directly in it. It returns `nil`,
the panic is not consumed, and unwinding continues.

Change it to defer `recoverPanic` directly:

```go
defer p.recoverPanic(&r)
```

Now `recoverPanic` itself is the deferred function. `recover()` is
called directly within it. The panic is consumed. `r.Err` is populated.
The batch continues with the next sample.

The argument `&r` is captured at defer time (defer evaluates its
function-call arguments immediately, even though the call itself is
deferred). Since `&r` is the address of a named return value whose
storage doesn't move, this is exactly what you want.

## A note on what's *not* the bug

The recovery logic inside `recoverPanic` is correct — it logs the
panic with a stack trace, sets `r.Err`, and returns. Don't be tempted
to rewrite it. The bug is purely about *where* `recover()` is invoked
relative to the `defer` statement, not what `recoverPanic` does once
it's called from the right place.
