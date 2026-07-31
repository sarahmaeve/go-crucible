package ingest_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-crucible/go-crucible/internal/ingest"
	"github.com/go-crucible/go-crucible/internal/types"
)

// TestExercise06_StuckPipeline verifies that ReadMetrics returns the caller's
// cancellation after its consumer stops accepting values.
func TestExercise06_StuckPipeline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	out := make(chan types.Metric) // unbuffered — consumer controls flow

	// InfiniteSource always has a metric ready, keeping the fixture focused on
	// whether downstream backpressure remains cancellable.
	src := ingest.NewInfiniteSource("cpu")
	done := make(chan error, 1)
	go func() {
		done <- ingest.ReadMetrics(ctx, src, out)
	}()

	// Establish that the stream is active, then exercise its cancellation path.
	<-out
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("ReadMetrics returned %v; want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ReadMetrics remained blocked on output after cancellation")
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
