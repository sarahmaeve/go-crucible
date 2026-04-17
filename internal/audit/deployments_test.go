package audit_test

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-crucible/go-crucible/internal/audit"
	"github.com/go-crucible/go-crucible/internal/client"
)

// TestExercise02_UnwrittenLabels verifies that AuditDeploymentLabels does not
// panic when a deployment is missing a required label. This test calls
// AuditDeploymentLabels directly (not via DeploymentAuditor).
func TestExercise02_UnwrittenLabels(t *testing.T) {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "api-server",
			Namespace: "default",
			// Intentionally no labels — all required labels will be missing.
			Labels: map[string]string{},
		},
	}

	fc := client.NewFakeClient(dep)

	// Use a deferred recover to catch any panic.
	panicked := false
	var panicVal any
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				panicVal = r
			}
		}()
		_, _ = audit.AuditDeploymentLabels(t.Context(), fc, "default", []string{"app", "version", "team"})
	}()

	if panicked {
		t.Errorf(
			"AuditDeploymentLabels panicked with: %v",
			panicVal,
		)
	}
}

// TestAuditDeploymentLabels_DetectsMissingLabels is a sanity-check test that
// verifies findings are generated for deployments missing required labels
// when the function works correctly.
func TestAuditDeploymentLabels_DetectsMissingLabels(t *testing.T) {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "frontend",
			Namespace: "production",
			Labels: map[string]string{
				"app": "frontend",
				// "version" and "team" are missing
			},
		},
	}

	fc := client.NewFakeClient(dep)

	// Wrap in recover so the test doesn't crash.
	var findings []interface{}
	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		f, err := audit.AuditDeploymentLabels(t.Context(), fc, "production", []string{"app", "version", "team"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		for _, fi := range f {
			findings = append(findings, fi)
		}
	}()

	if panicked {
		// Already reported in the exercise test; skip assertion here.
		return
	}

	// Expect 2 findings: missing "version" and "team".
	if len(findings) != 2 {
		t.Errorf("expected 2 findings for 2 missing labels, got %d", len(findings))
	}
}
