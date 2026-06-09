# Sample Review — PR #229

This is one reasonable review of the alert-summaries PR. Yours will
differ in tone, phrasing, and which process concerns you raise. Compare
the *substance* (did you catch the discarded write-back and the
unguarded send, did you avoid flagging the pointer-map accumulation?)
rather than the wording.

## Overall assessment

**Request changes.**

The aggregation design is sound — pointer-valued accumulation, value
copies on the way out, deterministic streaming order. But the
`Truncated` feature this PR exists to provide is a silent no-op
(`markTruncated` modifies copies and discards them), and `Stream`'s
goroutine has no exit path when the consumer stops reading, despite its
doc comment promising one. Both are small fixes; both need tests in
this PR.

## Blockers

### 1. `markTruncated` modifies copies — `Truncated` is always zero

**`internal/alert/summary.go:52-58`**, severity: **major**.

```go
func markTruncated(summaries map[string]Summary) {
	for _, s := range summaries {
		if s.Count > len(s.Examples) {
			s.Truncated = s.Count - len(s.Examples)
		}
	}
}
```

`Summary` is a struct, and ranging over a `map[string]Summary` yields
a **copy** of each value. `s.Truncated = ...` updates the copy; nothing
writes the copy back to the map, so every summary the caller receives
has `Truncated == 0` regardless of how many alerts were omitted. The
"and N more" rendering — the PR description's stated purpose for this
field — will never display. No error, no panic; the feature just
silently doesn't exist.

**Suggested fix:** range with the key and write back:

```go
for key, s := range summaries {
	if s.Count > len(s.Examples) {
		s.Truncated = s.Count - len(s.Examples)
		summaries[key] = s
	}
}
```

Or simpler: drop `markTruncated` entirely and set `Truncated` inside
`Aggregate`'s conversion loop, where `out[team] = *s` is built and the
count and example length are both at hand.

### 2. `Stream`'s send has no exit path — one leaked goroutine per evaluation cycle

**`internal/alert/summary.go:64-75`**, severity: **major**.

```go
for _, t := range teams {
	slog.DebugContext(ctx, "streaming summary", "team", t)
	out <- summaries[t]
}
```

The doc comment promises "Cancel ctx to abandon delivery" — but `ctx`
is only used for the debug log line. The send is bare. If the consumer
stops reading partway through (notifier shut down, error path,
slow-consumer timeout), the goroutine blocks on `out <- summaries[t]`
forever, and `defer close(out)` never runs — so the consumer side never
sees a close either. In a daemon calling this once per evaluation
cycle, that is an unbounded goroutine leak with each leaked goroutine
pinning its summaries map.

**Suggested fix:** make the send honour the promise the doc comment
already makes:

```go
select {
case out <- summaries[t]:
case <-ctx.Done():
	return // defer close(out) still signals the consumer
}
```

## Suggestions

### Consider returning the goroutine's completion to the caller

**`internal/alert/summary.go:62`**, severity: **minor**.

`Stream` is fire-and-forget; callers cannot distinguish "delivered
everything" from "abandoned on cancellation." A `<-chan struct{}` done
signal (or just documenting that the channel close is the only
completion signal) would help the notifier integration in the
follow-up PR. Not a blocker.

## Questions

### Where do team-less alerts go?

**`internal/alert/summary.go:28`.** Alerts without a `team` label
aggregate under `""`. Is the empty string routed anywhere in the
notifier, or do unowned firing alerts silently vanish from
notifications entirely? Either answer can be right — a fallback
"unowned" team that pages a default channel, or deliberate exclusion —
but the choice should be explicit, documented on `Aggregate`, and
tested. Asking for intent, not blocking.

## Nits

None worth the author's time.

## Things I Verified

### The pointer-map accumulation is correct — don't confuse it with the value-copy bug

**`internal/alert/summary.go:26-41`.**

```go
acc := make(map[string]*Summary)
...
s.Count++
```

This loop *also* mutates something obtained from a map in a range
loop — but `acc` is a `map[string]*Summary`. `s` is a pointer; the
mutation goes through it to the stored value. Pointer-valued maps
don't need write-backs. (If you flagged this after learning the
value-copy trap: the rule is about *copies*, and a copied pointer
still points at the same struct.)

### The value-copy conversion is deliberate

**`internal/alert/summary.go:43-46`.** `out[team] = *s` converts the
internal pointer map to value copies before returning — so callers
can't mutate the aggregator's internals through shared pointers.
Reasonable defensive choice; it's also exactly what makes the
`markTruncated` bug above possible, which is worth a comment once
that bug is fixed.

### Deterministic order in `Stream`

**`internal/alert/summary.go:66-69`.** Collecting keys and sorting
before sending makes delivery order stable across runs — map iteration
order would not be. Matches the test-plan claim. Good.

## Process

### Neither blocker is visible to the included test — please extend it in this PR

**PR test plan + `internal/alert/summary_test.go`**, severity:
**major process concern**.

The included test is well-shaped (counts, firing-only filter) but its
fixture has **two** firing alerts on one team — below the
three-example cap, so `Truncated` is legitimately zero and the
write-back bug is undetectable. And `Stream` has no test at all; "wired
into the notifier on a dev branch" exercises the happy path only.

Two asks:

1. Add a fixture with four-plus firing alerts on one team and assert
   `got["platform"].Truncated == 1` (or however many beyond the cap) —
   this fails on the current code and locks in the fix.
2. Add a `Stream` cancellation test: buffered consumer reads one
   summary, cancels ctx, then asserts the channel closes within a
   timeout. This fails (hangs) on the current code.

---

## What this sample review is trying to model

- **Doc comments are claims to verify.** "Cancel ctx to abandon
  delivery" reads as reassurance; the review checks the send statement
  against it and finds the promise unimplemented. Comments reviewed as
  claims catch the gap between intent and code.
- **The rule learned precisely, not as a pattern-match.** Two loops
  mutate range values; one is a bug. The review names the
  distinction — value type vs pointer type — in "Things I Verified,"
  which is how a reviewer demonstrates understanding rather than
  reflex.
- **Connecting fixture size to blind spots.** "Tests exist" is not the
  question; *can this fixture express the failure?* is. Two alerts can
  never exceed a three-example cap, so the test was structurally
  incapable of catching blocker 1.
- **Proposing the cheaper fix.** For blocker 1 the review offers both
  the minimal write-back and the simpler restructuring (set the field
  where the value is built). Giving the author an easy path shortens
  the round trip.
