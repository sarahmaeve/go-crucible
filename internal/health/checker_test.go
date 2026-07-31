package health_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-crucible/go-crucible/internal/health"
)

// TestExercise10_HangingHealthCheck verifies that Check respects the caller's
// context deadline and returns promptly when the deadline is exceeded.
func TestExercise10_HangingHealthCheck(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		slowCheck := health.CheckFunc{
			Name: "slow-db",
			Fn: func(ctx context.Context) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(10 * time.Second):
					return nil
				}
			},
		}

		checker := health.NewHealthChecker([]health.CheckFunc{slowCheck})
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()

		start := time.Now()
		err := checker.Check(ctx)
		elapsed := time.Since(start)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Check error = %v, want context.DeadlineExceeded", err)
		}
		if elapsed != 500*time.Millisecond {
			t.Errorf("Check returned after %v fake time, want 500ms", elapsed)
		}
	})
}
