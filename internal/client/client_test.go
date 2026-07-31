package client_test

import (
	"testing"

	"github.com/go-crucible/go-crucible/internal/client"
)

// TestExercise05_NilCheckThatLies verifies that NewAuditClient with an invalid
// kubeconfig returns a usable nil interface or a non-nil error.
//
// The returned interface must be nil when construction fails; any non-nil
// interface violates the constructor contract, whether or not a method call
// happens to panic.
func TestExercise05_NilCheckThatLies(t *testing.T) {
	// Use a guaranteed-nonexistent kubeconfig path. An empty string would
	// fall back to ~/.kube/config or in-cluster config, which may succeed
	// on a developer machine and silently hide the issue.
	c, err := client.NewAuditClient("/tmp/go-crucible-nonexistent-kubeconfig-test")

	// When config loading fails, the function must return a non-nil error.
	if err == nil {
		t.Fatal("NewAuditClient with an invalid kubeconfig returned nil error; expected non-nil")
	}

	if c != nil {
		t.Errorf("NewAuditClient returned non-nil interface %T after construction failed; want nil", c)
	}
}
