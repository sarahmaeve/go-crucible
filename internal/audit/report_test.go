package audit_test

import (
	"context"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/go-crucible/go-crucible/internal/audit"
	"github.com/go-crucible/go-crucible/internal/client"
	"github.com/go-crucible/go-crucible/internal/types"
)

// makeConstantAuditor returns an AuditFunc that always returns n findings,
// regardless of the context, client, or namespace.
func makeConstantAuditor(n int, name string) audit.AuditFunc {
	return func(_ context.Context, _ client.AuditClient, _ string) ([]types.Finding, error) {
		findings := make([]types.Finding, n)
		for i := range findings {
			findings[i] = types.Finding{
				Resource:  "Pod",
				Namespace: "default",
				Name:      fmt.Sprintf("%s-finding-%d", name, i),
				Severity:  types.SeverityWarning,
				Message:   fmt.Sprintf("finding %d from %s", i, name),
			}
		}
		return findings, nil
	}
}

// makeNoPodLimitsPod creates a pod with no CPU or memory limits for n containers.
func makeNoPodLimitsPod(namespace, name string, containers int) *corev1.Pod {
	cs := make([]corev1.Container, containers)
	for i := range cs {
		cs[i] = corev1.Container{
			Name:  fmt.Sprintf("container-%d", i),
			Image: "nginx",
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("64Mi"),
				},
				// No Limits set.
			},
		}
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec:       corev1.PodSpec{Containers: cs},
	}
}

// TestExercise12_RaceReport verifies that ConcurrentAudit collects findings
// from all auditors without dropping any results. Run with -race to also detect
// data races on the shared findings slice:
//
//	go test -race -count=5 ./internal/audit/ -run TestExercise12
func TestExercise12_RaceReport(t *testing.T) {
	const (
		numAuditors    = 10
		findingsEach   = 5
		wantTotal      = numAuditors * findingsEach
	)

	auditors := make([]audit.AuditFunc, numAuditors)
	for i := range auditors {
		auditors[i] = makeConstantAuditor(findingsEach, fmt.Sprintf("auditor-%d", i))
	}

	fc := client.NewFakeClient()
	report, err := audit.ConcurrentAudit(t.Context(), auditors, fc, "default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Summary.Total != wantTotal {
		t.Errorf(
			"expected %d total findings from %d auditors × %d findings each, got %d",
			wantTotal, numAuditors, findingsEach, report.Summary.Total,
		)
	}
}

// TestExercise13_LostGoroutine verifies that ParallelAudit collects findings
// from ALL goroutines before returning. The test runs multiple iterations to
// surface non-deterministic ordering issues. Run with -race for best coverage:
//
//	go test -race -count=10 ./internal/audit/ -run TestExercise13
func TestExercise13_LostGoroutine(t *testing.T) {
	const (
		numAuditors  = 8
		findingsEach = 3
		wantTotal    = numAuditors * findingsEach
		iterations   = 50 // repeat to make the race observable
	)

	auditors := make([]audit.AuditFunc, numAuditors)
	for i := range auditors {
		auditors[i] = makeConstantAuditor(findingsEach, fmt.Sprintf("parallel-auditor-%d", i))
	}

	fc := client.NewFakeClient()

	for i := range iterations {
		report, err := audit.ParallelAudit(t.Context(), auditors, fc, "default")
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
		if report.Summary.Total != wantTotal {
			t.Errorf(
				"iteration %d: expected %d total findings from all goroutines, got %d",
				i, wantTotal, report.Summary.Total,
			)
			// Report first failure and stop to avoid log spam.
			return
		}
	}
}
