# Sample Review — PR #221

This is one reasonable review of the remote-write PR. Yours will differ
in tone, phrasing, and which process concerns you raise. Compare the
*substance* (did you trace the constructor's return value into main.go,
did you catch the context, did you avoid flagging the body drain?)
rather than the wording.

## Overall assessment

**Request changes.**

The feature shape is right — implementing the existing `MetricSink`
interface keeps the daemon decoupled from the transport, and best-effort
semantics are correctly carried through `forwardToRemote`. But the two
error paths in `NewRemoteSink` return a typed nil that defeats the
degraded-mode check in `main.go` — the exact failure this PR promises
to handle gracefully becomes a startup panic instead. And `Publish`
ignores the context it is handed, so shutdown can't cancel in-flight
pushes. Both are small fixes. I'd also like the two tests that would
have caught them to land in this PR.

## Blockers

### 1. `NewRemoteSink` returns a typed nil — degraded mode panics instead of degrading

**`internal/ingest/remote.go:28-34` + `cmd/pipeline/main.go` (the
`if sink != nil` block)**, severity: **major**.

```go
return (*RemoteSink)(nil), fmt.Errorf("remote sink: invalid base URL %q: %w", baseURL, err)
```

The function's return type is the `MetricSink` interface. Returning
`(*RemoteSink)(nil)` stores (type=`*RemoteSink`, value=nil) in that
interface — and an interface holding a typed nil is **not** `nil`.
Walk the caller:

```go
sink, err = ingest.NewRemoteSink(*remoteURLFlag, 5*time.Second)
if err != nil {
    slog.Warn("remote write disabled: invalid -remote-url", "err", err)
}
if sink != nil {
    slog.Info("remote write enabled", "url", *remoteURLFlag)
}
```

With a misconfigured URL — say `-remote-url collector.internal:8080`
(missing scheme, so `url.Parse` yields scheme `"collector.internal"`
and the scheme check fires) — the operator sees *both* log lines:
"remote write disabled" followed by "remote write enabled." Then the
typed-nil sink is passed into `RunPipeline`, the `sink != nil` check
there passes too, and the first `sink.Publish(...)` calls a method on
a nil `*RemoteSink` receiver: nil-pointer panic, daemon down. The PR's
own motivation section says collector problems "must not block local
processing" — this turns a config typo into a crash loop.

**Suggested fix:** return untyped nil on **both** error paths:

```go
return nil, fmt.Errorf("remote sink: invalid base URL %q: %w", baseURL, err)
```

Check both sites — the scheme-check path has the same problem. (And in
`main.go`, consider making the two conditions one `if err != nil { ... } else { ... }`
so "disabled" and "enabled" can never both log.)

### 2. `Publish` builds its request from `context.Background()`, not `ctx`

**`internal/ingest/remote.go:44`**, severity: **major**.

```go
req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, s.endpoint, bytes.NewReader(body))
```

`Publish` accepts a `ctx context.Context` — that's the `MetricSink`
contract, and `forwardToRemote` dutifully passes the pipeline's
shutdown context — but the request is built from a fresh
`context.Background()`. Consequences:

- SIGTERM cancels the pipeline context, but an in-flight push keeps
  going; shutdown waits on it (or abandons it mid-write).
- Any per-request deadline or tracing metadata a future caller attaches
  to `ctx` silently never reaches the HTTP layer.

To be fair about severity: this is not unbounded — the
`http.Client{Timeout: timeout}` still caps each request at 5 seconds.
But that cap is a backstop, not a substitute for cancellation, and the
first person to "clean up" the client timeout removes the only bound.

**Suggested fix:** `http.NewRequestWithContext(ctx, ...)`.

## Suggestions

### Rate-limit the failure log in `forwardToRemote`

**`cmd/pipeline/main.go`, `forwardToRemote`**, severity: **minor**.

During a collector outage, every metric logs a `Warn` — at the fake
source's rate that's a line per metric, and in production it's a log
flood that can cost more than the outage. Consider logging the first
failure and then a sampled or periodic summary ("remote publish
failing for 4m32s, 27,012 dropped"). Not a blocker for a first cut.

## Questions

### Should non-2xx classification distinguish permanent from transient?

**`internal/ingest/remote.go:56-58`.** A `401 Unauthorized` and a
`503 Service Unavailable` currently produce the same outcome: one
warning line, metric dropped. For best-effort forwarding that may be
exactly right — but a misconfigured credential will silently drop
every metric forever, distinguishable from a transient blip only by
reading timestamps. Is that acceptable for the alert-routing use case
this feeds? If not, a follow-up that treats 4xx as "stop and complain
loudly" is worth a ticket. Asking, not blocking.

## Nits

None worth the author's time.

## Things I Verified

### The body drain before close is correct, not dead code

**`internal/ingest/remote.go:53-54`.**

```go
defer resp.Body.Close()

// Drain the body so the underlying connection can be reused.
_, _ = io.Copy(io.Discard, resp.Body)
```

At first glance this reads as "copy a response we never use — why?"
It's the keep-alive idiom: an HTTP/1.1 connection only returns to the
client's connection pool if the response body has been fully read;
closing an undrained body tears the connection down and forces a new
TCP (and possibly TLS) handshake per publish. Drain-then-close is
exactly right for a hot path that pushes every metric. Verified, not
flagged — and the comment the author left is appreciated.

### `u.JoinPath` endpoint construction

**`internal/ingest/remote.go:36`.** `JoinPath("/api/v1/push")` handles
trailing slashes on the base URL correctly (no `//api/v1/push`), so
both `http://collector:8080` and `http://collector:8080/` work. Fine.

## Process

### Both blockers are invisible to the included test — please add the two tests that see them

**PR test plan + `internal/ingest/remote_test.go`**, severity:
**major process concern**.

The included unit test is good as far as it goes, but it constructs
the sink from a *valid* `httptest.Server` URL and publishes with
`context.Background()` — the one configuration in which neither bug
can fire. Two additions would have caught both blockers and will keep
them fixed:

1. **Constructor error path:** `NewRemoteSink("collector.internal:8080", time.Second)`
   must return an error **and** a sink that compares equal to `nil` —
   assert `sink == nil` explicitly; that assertion is precisely what
   distinguishes untyped nil from the typed-nil bug.
2. **Cancellation:** against an `httptest.Server` whose handler blocks
   on a channel, call `Publish` with an already-cancelled context and
   assert it returns promptly with a context error rather than riding
   out the full client timeout.

---

## What this sample review is trying to model

- **Tracing a value across files.** Neither blocker is visible inside
  one hunk. The review connects `remote.go`'s return statements to
  `main.go`'s nil checks, and `Publish`'s signature to what
  `forwardToRemote` passes in. Intermediate-tier review is exactly
  this: simulating the runtime, not pattern-matching lines.
- **Connecting a bug to the PR's own goals.** "Typed nil defeats a nil
  check" is the language lesson; "your degraded-mode requirement
  becomes a crash loop" is the review. The second framing is what gets
  bugs fixed without an argument.
- **Honest severity on the context bug.** The client timeout genuinely
  bounds the damage, and the review says so before explaining why the
  fix still matters. Overclaiming ("this hangs forever!") costs
  credibility that the next review pays for.
- **Verifying the idiom that looks wrong.** The body drain is the kind
  of line a new reviewer flags as waste. Naming *why* it's correct —
  connection reuse — demonstrates the review actually engaged with the
  code.
- **Asking for the tests that map to the bugs.** Not "more tests
  please" but the two specific tests whose assertions correspond to
  the two blockers.
