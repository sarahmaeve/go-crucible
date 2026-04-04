package ingest_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/go-crucible/go-crucible/internal/ingest"
	"github.com/go-crucible/go-crucible/internal/types"
)

// TestExercise06_StuckPipeline verifies that ReadMetrics does not leak
// goroutines after the context is cancelled.
func TestExercise06_StuckPipeline(t *testing.T) {
	baseline := runtime.NumGoroutine()

	ctx, cancel := context.WithCancel(context.Background())
	out := make(chan types.Metric) // unbuffered — consumer controls flow

	// InfiniteSource always has a metric ready; it never blocks on Read.
	// This means the goroutine inside ReadMetrics will immediately try to
	// send to out. Once we cancel and stop consuming, the goroutine will be
	// stuck on `out <- m` with no way to escape.
	src := ingest.NewInfiniteSource("cpu")
	if err := ingest.ReadMetrics(ctx, src, out); err != nil {
		t.Fatalf("ReadMetrics returned unexpected error: %v", err)
	}

	// Consume one metric to let the goroutine start, then cancel and abandon
	// the channel.
	<-out
	cancel()

	// Give the goroutine time to exit (if it respects ctx.Done).
	time.Sleep(200 * time.Millisecond)

	after := runtime.NumGoroutine()
	// After cancellation the ReadMetrics goroutine should have exited, so we
	// expect to be back at (roughly) the baseline. Allow +1 for transient
	// runtime goroutines but NOT for the leaked reader goroutine.
	if after > baseline {
		t.Errorf("goroutine leak detected — baseline %d, after cancel %d (want <= %d)",
			baseline, after, baseline)
	}
}

// TestExercise14_ForeverForwarder verifies that ForwardMetrics returns when
// its input channel is closed.
func TestExercise14_ForeverForwarder(t *testing.T) {
	ctx := context.Background()
	in := make(chan types.Metric, 4)
	out := make(chan types.Metric, 4)

	// Pre-fill and then close the input channel.
	in <- types.Metric{Name: "cpu", Value: 1.0}
	in <- types.Metric{Name: "cpu", Value: 2.0}
	close(in)

	done := make(chan error, 1)
	go func() {
		done <- ingest.ForwardMetrics(ctx, in, out)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("exercise 14: ForwardMetrics returned unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("exercise 14: ForwardMetrics did not return after input channel was closed (goroutine leak / infinite loop)")
	}
}
