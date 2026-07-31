package health_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-crucible/go-crucible/internal/health"
)

// TestExercise10_HangingHealthCheck verifies that Check respects the caller's
// context deadline and returns promptly when the deadline is exceeded.
func TestExercise10_HangingHealthCheck(t *testing.T) {
	// A slow dependency that only returns when its context is cancelled.
	slowCheck := health.CheckFunc{
		Name: "slow-db",
		Fn: func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(10 * time.Second): // intentionally long
				return nil
			}
		},
	}

	checker := health.NewHealthChecker([]health.CheckFunc{slowCheck})

	// Give the check only 500 ms to complete.
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	start := time.Now()
	go func() {
		done <- checker.Check(ctx)
	}()

	select {
	case err := <-done:
		elapsed := time.Since(start)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Check error = %v, want context.DeadlineExceeded", err)
		}
		// Guard against an implementation that returns a fabricated error
		// immediately without running the dependency until the deadline.
		if elapsed < 400*time.Millisecond {
			t.Errorf("Check returned after %v, before the 500ms caller deadline", elapsed)
		}
	case <-time.After(1 * time.Second):
		t.Error("exercise 10: Check did not return within the context deadline — health check ignored cancellation")
	}
}
