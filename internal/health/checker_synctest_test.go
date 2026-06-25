//go:build synctest

// This file is an EXTENSION to exercise 10 (see
// exercises/10-hanging-health-check/EXTENSION.md and docs/synctest.md). It is
// gated behind the `synctest` build tag so it never runs in the canonical suite
// (`go test ./...`, `make status`, `make verify`). Run it deliberately:
//
//	go test -tags synctest ./internal/health/ -run TestExercise10_Synctest -v
//
// `synctest` here is an ordinary Go build tag, NOT GOEXPERIMENT — the
// testing/synctest package graduated to the standard library in Go 1.25.

package health_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/go-crucible/go-crucible/internal/health"
)

// TestExercise10_Synctest is the testing/synctest rewrite of the context-deadline
// check. Compare it against TestExercise10_HangingHealthCheck in checker_test.go,
// which spawns a goroutine, races a real 500ms deadline against a real 1s
// wall-clock timeout, and can only assert the weak proxy "Check returned within
// a second" — taking ~0.5-1s of real time to do so.
//
// Inside a synctest bubble the clock is fake. The 500ms deadline and the
// dependency's 10s timer both resolve in zero real time, and time.Since reports
// fake-elapsed duration. That lets the test assert the EXACT property exercise 10
// teaches — Check honored the caller's deadline — instead of a timing proxy:
//
//   - on the FIXED Check (passes ctx), the dependency's select observes
//     ctx.Done() at fake t=500ms and returns context.DeadlineExceeded.
//   - on the BUGGY Check (context.Background()), the deadline never reaches the
//     dependency; it returns nil only when its own 10s timer fires at fake
//     t=10s. Both assertions below then fail — instantly and deterministically.
//
// Exercise 10 is pre-solved on main, so this passes as-is. To watch it fail,
// reintroduce the bug first: git apply -R solutions/10-hanging-health-check.patch
func TestExercise10_Synctest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// A slow dependency that only returns when its context is cancelled, or
		// after an intentionally long fake-time delay.
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
		elapsed := time.Since(start) // fake-elapsed duration inside the bubble

		if elapsed >= 10*time.Second {
			t.Errorf("Check ignored the caller deadline: returned after %v (fake) with err=%v", elapsed, err)
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("want context.DeadlineExceeded, got %v after %v (fake)", err, elapsed)
		}
	})
}
