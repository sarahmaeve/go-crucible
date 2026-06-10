# Diagnosis-Track Artifacts — Capture Recipes and Format Notes

The `ARTIFACT.txt` files in `exercises/diagnosis/` are curated:
hostnames, paths, addresses, and goroutine numbers are fictional, but
the **structure** of each artifact was validated against real tool
output (2026-06-09, go 1.26.3, darwin/amd64) and every in-repo
`file:line` is pinned in the registry (`references` fields, enforced
by `make verify-quick`).

This note records how to re-capture each artifact type — for
re-validating after a toolchain upgrade, and for building new
diagnosis exercises from captures rather than from memory. **Always
capture first, then fictionalise**: writing tool output from memory
gets the shape subtly wrong (see "what the captures corrected" below).

When fictionalising a capture: keep every structural element (frame
order, indentation, header wording, `+0x` offsets present, blank-line
placement); replace paths, hostnames, addresses, and goroutine
numbers with in-universe values; keep one real-looking offset style
(`+0x52`, not `+0xABCDEF`).

## D01 — goroutine profile (pprof debug=1 / debug=2)

Throwaway harness (delete after capture; it must live inside the repo
to import `internal/`):

```go
package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/go-crucible/go-crucible/internal/ingest"
	"github.com/go-crucible/go-crucible/internal/types"
)

func main() {
	for i := 0; i < 50; i++ {
		src := ingest.NewFakeSourceN(fmt.Sprintf("target-%d", i), 1.0, 100)
		out := make(chan types.Metric) // nobody ever reads
		_ = ingest.ReadMetrics(context.Background(), src, out)
	}

	time.Sleep(2 * time.Second)
	runtime.GC() // see the waitsince note below
	time.Sleep(75 * time.Second)
	runtime.GC()

	p := pprof.Lookup("goroutine")
	f1, _ := os.Create("goroutine-debug1.txt")
	_ = p.WriteTo(f1, 1)
	f1.Close()
	f2, _ := os.Create("goroutine-debug2.txt")
	_ = p.WriteTo(f2, 2)
	f2.Close()
}
```

Format facts (validated):

- **debug=1** buckets symbolise the user frames (and `runtime.main`)
  but NOT the parking machinery — the leaked bucket is 5 raw PCs in
  the `@` line and exactly one `#` line for `ReadMetrics.func1`.
  `created by` does not appear in debug=1 at all.
- **debug=2 wait durations** (`[chan send, 1129 minutes]`) only
  appear once the runtime has stamped the goroutine's `waitsince`,
  which happens during GC scans — an idle harness that never
  allocates never GCs, and the durations silently don't show. Force
  `runtime.GC()` after the goroutines park (and note durations under
  one minute never show). Live services GC constantly, so production
  dumps always have them. The format is `N minutes` — never
  singularised (`1 minutes` is what the runtime prints).
- **No `gowrap` frame** for `reader.go:12`'s `go func() {...}()` —
  wrapper frames (`X.gowrap1`) are only generated when the `go`
  statement passes arguments (compare D02).

## D02 — race detector report

```bash
go test -race ./internal/audit/ -run TestExercise12 -count=5 2>&1 | head -80
go test -race ./internal/audit/ -run TestExercise12 -count=1 -v 2>&1 | tail -12
```

Format facts (validated):

- The first reported pair for the append race is **`Read at` /
  `Previous write at`** (append's header load racing another append's
  store). Later blocks in the same run show write/write pairs, and
  one pair surfaces inside `runtime.growslice`/`runtime.slicecopy`.
  Which shape comes first is scheduling luck.
- The same source line carries **two different `+0x` offsets** in the
  two stacks (load vs store instruction) — a teaching point in the
  exercise.
- Each access stack includes a **`ConcurrentAudit.gowrap1` frame at
  `report.go:56`** — the compiler-generated wrapper for the
  argument-passing `go` statement (Go 1.22+).
- `created at` stacks are four frames: the creating function, the
  test function, `testing.tRunner`, `testing.(*T).Run.gowrap1`.
- The failure tail cites the **opening line of a multi-line
  `t.Errorf` call** (`report_test.go:81`), and
  `testing.go:<line>: race detected during execution of test` — the
  testing.go line number is toolchain-specific (1712 on go 1.26.3).

## D03 — panic traceback

Throwaway harness: a `main` that builds a `worker.NewPool` with a
processor that panics on one input, then calls `Process` on a
2,000-element batch (see the D03 README's in-universe `reprocess`
tool; the harness is the same shape, ~35 lines).

Format facts (validated):

- **Modern Go does NOT print a `panic({...})` runtime frame** — the
  traceback starts at the panicking function. (Older artifacts and
  blog posts show `panic(...)` at `runtime/panic.go:NNN`; do not
  reproduce that from memory.)
- Frame arguments render as **expanded struct shapes**: a
  `types.Metric` appears as
  `{{0x..., 0x11}, 0x0, 0x405d400000000000, {0x0, 0x0, 0x0}}` —
  string header, nil map, float bits (0x405d40... is 117.0), zero
  time. `?` suffixes mark values the runtime recovered from
  registers rather than stack slots.
- `exit status 2` is what `go run` prints; a job runner capturing a
  compiled binary's stderr sees the shell's exit-status report
  instead — the artifact keeps the line as runner output.

## What the captures corrected (2026-06-09)

For the record, hand-authoring got these wrong before validation:
D02 claimed a Write/Write first pair, omitted both `gowrap1` frames
and the fourth created-at frame, and invented `testing.go:1490`; D03
included a nonexistent `panic({...})` runtime frame; D01's frame
offsets were arbitrary. All structural; all invisible until compared
against real output — which is exactly why this note exists.
