# Hints for Review Exercise 04

These hints are progressive — read one at a time, try the review again,
and only open the next hint if you're still stuck.

## Hint 1: Count

There are **two** correctness issues in this diff that must be fixed
before merging. There is **one** suspicious-looking line that is
actually correct and belongs in "Things I Verified," **one** process
concern about the included unit test, and **one** question about intent
worth raising.

Both correctness issues share a property: nothing in the hunk where
they live looks alarming on its own. You have to simulate what happens
at runtime — in one case across two files.

## Hint 2: Categories

Without naming lines, the two correctness issues are:

1. A constructor's error paths return a value that defeats the
   caller's nil check. The cause lives in one file and the consequence
   in another: trace what `NewRemoteSink` actually returns into what
   `cmd/pipeline/main.go` does with it. This is Exercise 05's lesson
   in diff form — and note that it directly contradicts the PR's own
   stated design goal of degraded mode.

2. A method receives a perfectly good context from its caller and then
   ignores it at the one call where it matters. Compare the
   `MetricSink.Publish` contract (context-first parameter) with what
   the HTTP request is actually built from. This is Exercise 10's
   lesson in diff form.

The line you should verify-but-not-flag involves deliberately throwing
bytes away.

## Hint 3: Lines

- `internal/ingest/remote.go`, `NewRemoteSink`: both `return`
  statements in the error paths return `(*RemoteSink)(nil)` as the
  `MetricSink` interface. An interface holding (type=`*RemoteSink`,
  value=nil) is **not** equal to `nil`. So in `cmd/pipeline/main.go`,
  after a bad URL: the `err != nil` branch logs "remote write
  disabled" — and then `if sink != nil` passes anyway, logs "remote
  write enabled," and the first `sink.Publish(...)` dereferences a nil
  receiver and panics. Both contradictory log lines appear, then the
  daemon dies. The fix: `return nil, fmt.Errorf(...)` — untyped nil —
  on **both** error paths.

- `internal/ingest/remote.go`, `Publish`: the request is built with
  `http.NewRequestWithContext(context.Background(), ...)`. The caller's
  `ctx` — which carries pipeline shutdown — never reaches the request.
  The only remaining bound is the client's flat 5-second timeout. Fix:
  pass `ctx`.

- The `io.Copy(io.Discard, resp.Body)` line is the connection-reuse
  idiom: an HTTP/1.1 connection only returns to the keep-alive pool
  once the response body has been fully read. It is correct. Verify it,
  write it down, move on.

- The test plan: which of the two bugs would any test in this PR
  catch? (Neither — the test uses a valid URL and a Background
  context.) Ask for an invalid-URL constructor test asserting
  `sink == nil`, and a cancelled-context `Publish` test against a
  server that blocks.
