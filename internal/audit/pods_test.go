package audit_test

import (
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-crucible/go-crucible/internal/audit"
	"github.com/go-crucible/go-crucible/internal/client"
)

// TestExercise01_SilentFailure verifies that AuditPodLimits propagates errors
// from the underlying client rather than returning zero findings.
func TestExercise01_SilentFailure(t *testing.T) {
	sentinelErr := errors.New("simulated API failure")

	errClient := &client.ErrorClient{
		PodError: sentinelErr,
	}

	findings, err := audit.AuditPodLimits(t.Context(), errClient, "default")

	if err == nil {
		t.Errorf(
			"AuditPodLimits returned nil error when ListPods failed with %q; expected error propagation",
			sentinelErr,
		)
	}

	if len(findings) > 0 {
		t.Errorf(
			"expected 0 findings when client errors, got %d",
			len(findings),
		)
	}
}

// TestAuditPodLimits_FindsMissingLimits is a sanity-check (non-exercise) test
// that confirms the happy path: pods without resource limits produce findings.
func TestAuditPodLimits_FindsMissingLimits(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "no-limits-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx",
					Resources: corev1.ResourceRequirements{
						// No limits set
					},
				},
			},
		},
	}

	fc := client.NewFakeClient(pod)
	findings, err := audit.AuditPodLimits(t.Context(), fc, "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) == 0 {
		t.Error("expected findings for pod with missing resource limits, got none")
	}
}

// TestAuditPodLimits_NoFindingsWhenLimitsSet confirms pods with proper limits
// produce no findings.
func TestAuditPodLimits_NoFindingsWhenLimitsSet(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "good-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: "nginx",
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
					},
				},
			},
		},
	}

	fc := client.NewFakeClient(pod)
	findings, err := audit.AuditPodLimits(t.Context(), fc, "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected 0 findings for pod with proper limits, got %d", len(findings))
	}
}
