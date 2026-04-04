package client_test

import (
	"testing"

	"github.com/go-crucible/go-crucible/internal/client"
)

// TestExercise05_NilCheckThatLies verifies that NewAuditClient with an invalid
// kubeconfig returns a usable nil interface or a non-nil error.
//
// The test exposes two facets of the expected behavior:
//  1. Checks that the returned interface is nil when an error occurs.
//  2. Attempts to call ListPods on the returned client to confirm it does not
//     panic and properly propagates errors.
func TestExercise05_NilCheckThatLies(t *testing.T) {
	// Use a guaranteed-nonexistent kubeconfig path. An empty string would
	// fall back to ~/.kube/config or in-cluster config, which may succeed
	// on a developer machine and silently hide the issue.
	c, err := client.NewAuditClient("/tmp/go-crucible-nonexistent-kubeconfig-test")

	// When config loading fails, the function must return a non-nil error.
	if err == nil {
		t.Fatal("NewAuditClient with an invalid kubeconfig returned nil error; expected non-nil")
	}

	// A nil interface is the correct return value when an error occurs.
	if c == nil {
		return // correct behavior
	}

	// The interface is non-nil — check whether it panics on use.
	panicked := func() (didPanic bool) {
		defer func() {
			if r := recover(); r != nil {
				didPanic = true
			}
		}()
		_, _ = c.ListPods("default") //nolint:errcheck
		return false
	}()

	if panicked {
		t.Errorf("NewAuditClient returned a non-nil client that panics on use")
	}
}
