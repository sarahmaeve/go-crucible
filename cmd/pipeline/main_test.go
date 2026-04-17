package main

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/go-crucible/go-crucible/internal/ingest"
	"github.com/go-crucible/go-crucible/internal/types"
)

// TestExercise19_GracelessShutdown verifies that RunPipeline shuts down
// cleanly. It checks three independent behaviors, each in its own subtest,
// so that all must pass for the exercise to be considered complete.
func TestExercise19_GracelessShutdown(t *testing.T) {
	t.Run("bug19-2_double_close_panic", func(t *testing.T) {
		runOnce := func(label string) (err error) {
			src := ingest.NewFakeSource([]types.Metric{
				{Name: "test.metric", Value: 1.0, Labels: map[string]string{}},
			})

			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			defer cancel()

			done := make(chan error, 1)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						done <- fmt.Errorf("exercise 19 [%s]: RunPipeline panicked — deferred close on already-closed channel: %v", label, r)
					}
				}()
				done <- RunPipeline(ctx, []ingest.MetricSource{src})
			}()

			select {
			case e := <-done:
				return e
			case <-time.After(2 * time.Second):
				return fmt.Errorf("exercise 19 [%s]: RunPipeline did not return within 2 seconds after context cancellation", label)
			}
		}

		// First call: should return cleanly.
		if err := runOnce("call-1"); err != nil {
			if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
				t.Errorf("first call: %v", err)
			}
		}

		// Second call: should also return cleanly without panicking.
		if err := runOnce("call-2"); err != nil {
			t.Errorf("second call: %v", err)
		}
	})

	// Note: the old "signal_notify_called" subtest was removed when main()
	// migrated to signal.NotifyContext. That helper registers signals and
	// creates the context in one structural step, so the failure mode the
	// subtest guarded against — registering a channel but forgetting to call
	// signal.Notify — is no longer expressible. See .crucible/notes/19.md.

	t.Run("goroutine_exits_on_context_cancellation", func(t *testing.T) {
		// Verify that background goroutines spawned by RunPipeline exit when
		// the pipeline's context is cancelled. We use a BlockingSource that
		// only unblocks when its context is done, then count goroutines after
		// cancellation to check for leaks.

		baseline := runtime.NumGoroutine()

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		// Reset doneCh to avoid interference from bug 19-2.
		doneCh = make(chan struct{})

		done := make(chan error, 1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					done <- fmt.Errorf("panic: %v", r)
					return
				}
			}()
			// Use a BlockingSource that only unblocks when ctx is done.
			src := &ingest.BlockingSource{}
			done <- RunPipeline(ctx, []ingest.MetricSource{src})
		}()

		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatal("exercise 19: RunPipeline did not return within 3 seconds")
		}

		// Wait for goroutines to settle.
		time.Sleep(300 * time.Millisecond)

		leaked := runtime.NumGoroutine() - baseline
		if leaked > 0 {
			t.Errorf("exercise 19: %d goroutine(s) still running after context cancellation", leaked)
		}
	})
}
