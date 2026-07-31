//go:build synctest

// This file is an EXTENSION to exercise 06 (see
// exercises/06-stuck-pipeline/EXTENSION.md and docs/synctest.md). It is gated
// behind the `synctest` build tag so it never runs in the canonical suite
// (`go test ./...`, `make status`, `make verify`). Run it deliberately:
//
//	go test -tags synctest ./internal/ingest/ -run TestExercise06_Synctest -v
//
// `synctest` here is an ordinary Go build tag, NOT GOEXPERIMENT — the
// testing/synctest package graduated to the standard library in Go 1.25.

package ingest_test

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/go-crucible/go-crucible/internal/ingest"
	"github.com/go-crucible/go-crucible/internal/types"
)

// TestExercise06_Synctest is the testing/synctest rewrite of the goroutine-leak
// check. Compare it against TestExercise06_StuckPipeline in reader_test.go,
// which samples runtime.NumGoroutine() before and after a 200ms time.Sleep and
// reports a leak as a count that drifted ("baseline 2, after cancel 3").
//
// This version runs the code inside a synctest bubble. There is no sleep and no
// goroutine counting. After cancellation, synctest.Wait blocks until every
// other goroutine in the bubble is durably blocked or has exited. A
// cancellation-aware reader drains the bubble; one that remains blocked after
// cancellation produces a deterministic deadlock report with the blocked frame.
func TestExercise06_Synctest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		out := make(chan types.Metric) // unbuffered — the consumer controls flow

		// InfiniteSource keeps the fixture focused on downstream backpressure.
		src := ingest.NewInfiniteSource("cpu")
		if err := ingest.ReadMetrics(ctx, src, out); err != nil {
			t.Fatalf("ReadMetrics returned unexpected error: %v", err)
		}

		<-out    // consume one metric so the goroutine is running
		cancel() // cancel and stop consuming

		// Let every other bubble goroutine reach a durable block or exit.
		synctest.Wait()
	})
}
