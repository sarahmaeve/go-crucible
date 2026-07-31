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

// TestExercise06_Synctest expresses the same lifecycle contract as the
// canonical test inside a synctest bubble. This version replaces its bounded
// real-time timeout with deterministic coordination.
func TestExercise06_Synctest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		out := make(chan types.Metric)

		// Use a predictable source that always has another value available.
		src := ingest.NewInfiniteSource("cpu")
		done := make(chan error, 1)
		go func() {
			done <- ingest.ReadMetrics(ctx, src, out)
		}()

		<-out
		cancel()

		// Let every other bubble goroutine reach a durable block or exit.
		synctest.Wait()
		if err := <-done; err != context.Canceled {
			t.Fatalf("ReadMetrics returned %v; want context.Canceled", err)
		}
	})
}
