package ingest_test

import (
	"context"
	"errors"
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

	// InfiniteSource always has a metric ready, keeping the fixture focused on
	// whether downstream backpressure remains cancellable.
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
	// After cancellation the ReadMetrics goroutine should have exited. Allow
	// only the baseline runtime goroutines.
	if after > baseline {
		t.Errorf("goroutine leak detected — baseline %d, after cancel %d (want <= %d)",
			baseline, after, baseline)
	}
}

// TestExercise14_ForeverForwarder verifies both exit paths promised by
// ForwardMetrics: a closed input ends the stream after forwarding all received
// metrics, and cancellation interrupts a blocked downstream send.
func TestExercise14_ForeverForwarder(t *testing.T) {
	t.Run("closed input forwards all metrics and returns", func(t *testing.T) {
		in := make(chan types.Metric, 2)
		out := make(chan types.Metric, 2)
		in <- types.Metric{Name: "cpu", Value: 1.0}
		in <- types.Metric{Name: "cpu", Value: 2.0}
		close(in)

		done := make(chan error, 1)
		go func() {
			done <- ingest.ForwardMetrics(t.Context(), in, out)
		}()

		select {
		case err := <-done:
			if err != nil {
				t.Errorf("ForwardMetrics returned unexpected error: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("ForwardMetrics did not return after input channel was closed")
		}

		for i, want := range []float64{1.0, 2.0} {
			select {
			case got := <-out:
				if got.Value != want {
					t.Errorf("output[%d].Value = %v, want %v", i, got.Value, want)
				}
			default:
				t.Errorf("output[%d] missing; ForwardMetrics returned before forwarding all input", i)
			}
		}
	})

	t.Run("cancellation interrupts blocked output", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		in := make(chan types.Metric)
		out := make(chan types.Metric) // deliberately has no receiver
		done := make(chan error, 1)

		go func() {
			done <- ingest.ForwardMetrics(ctx, in, out)
		}()

		// This handshake ensures ForwardMetrics has accepted the metric before
		// cancellation is issued while downstream remains unavailable.
		in <- types.Metric{Name: "cpu", Value: 1.0}
		cancel()

		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("ForwardMetrics error = %v, want context.Canceled", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("ForwardMetrics remained blocked on output after cancellation")
		}
	})
}
