# Exercise 21: The Unbounded Request

**Application:** pipeline | **Difficulty:** Intermediate

## Symptoms

`PushHandler` accepts JSON metric batches over HTTP. The constructor takes
a `maxBytes` argument that is documented as "the per-request body-size
cap" and is stored on the handler struct. In practice, the handler happily
accepts and decodes request bodies of any size — the limit is never
enforced. A misbehaving or hostile client can post an arbitrarily large
payload; the handler reads it all into memory before even noticing.

## Reproduce

```bash
go test ./internal/ingest/ -run TestExercise21 -v
```

The exercise test configures the handler with a 512-byte limit and posts
a well-formed JSON body of roughly 4 KB. The body is valid JSON — the
oversize is the issue, not malformed content. The handler should reject
the request with 413 Request Entity Too Large; it currently returns 200
and publishes the metric anyway.

A companion happy-path test (`TestPushHandlerHappyPath`) confirms the
handler continues to work correctly for requests below the limit. That
test must remain green both before and after the fix.

## File to Investigate

`internal/ingest/http.go` — look at `PushHandler.ServeHTTP` and its
handling of the request body. The struct has a `maxBytes` field; trace
where it is read inside `ServeHTTP`.

## What You Will Learn

- `net/http` does not cap request-body size by default. A handler that
  calls `json.NewDecoder(r.Body).Decode(...)` on unbounded input will
  allocate memory proportional to whatever the client chose to send.
- `http.MaxBytesReader(w, r.Body, n)` wraps the body so that reads past
  `n` return `*http.MaxBytesError`. Assigning the wrapped reader back to
  `r.Body` is the canonical pattern, so any subsequent `io.Reader` user
  (the JSON decoder, a streaming parser, `io.Copy`) inherits the cap.
- When the decoder returns an error, classify it: a `*http.MaxBytesError`
  deserves a `413 Request Entity Too Large`, while malformed JSON
  deserves a `400 Bad Request`. Collapsing both into one response hides
  an important signal from operators and clients.
- Defense in depth: apply the cap **before** decoding, not after. A fix
  that decodes into memory and then checks the size would still allow
  the memory exhaustion.

## Related Exercises

- [Exercise 09: The Immortal Connection](../09-immortal-connection/README.md)
  — the other side of defensive HTTP handling: always close what you open.
- [Exercise 10: The Hanging Health Check](../10-hanging-health-check/README.md)
  — context propagation inside an HTTP handler. Together these three
  exercises form the core of "what a Go HTTP handler must get right."

## Fixing It

Apply your fix, then run:

```bash
go test ./internal/ingest/ -run TestExercise21 -v
go test ./internal/ingest/ -run TestPushHandlerHappyPath -v
```

Both tests must pass. See [HINTS.md](./HINTS.md) for progressive hints
if you get stuck.
