// Package audit implements Kubernetes resource auditing functions.
package audit

import (
	"context"
	"log/slog"

	corev1 "k8s.io/api/core/v1"

	"github.com/go-crucible/go-crucible/internal/client"
	"github.com/go-crucible/go-crucible/internal/types"
)

// AuditPodLimits inspects every pod in namespace and returns a Finding for
// each container that is missing CPU or memory resource limits.
func AuditPodLimits(ctx context.Context, c client.AuditClient, namespace string) ([]types.Finding, error) {
	pods, err := c.ListPods(ctx, namespace)
	if err != nil {
		slog.Error("AuditPodLimits: failed to list pods", "err", err)
	}

	var findings []types.Finding
	for _, pod := range pods {
		findings = append(findings, auditPodContainerLimits(pod)...)
	}
	return findings, nil
}

func auditPodContainerLimits(pod corev1.Pod) []types.Finding {
	var findings []types.Finding
	for _, c := range pod.Spec.Containers {
		if !hasLimit(c.Resources.Limits, corev1.ResourceCPU) {
			findings = append(findings, types.Finding{
				Resource:  "Pod",
				Namespace: pod.Namespace,
				Name:      pod.Name,
				Severity:  types.SeverityWarning,
				Message:   "container " + c.Name + " is missing a CPU limit",
			})
		}
		if !hasLimit(c.Resources.Limits, corev1.ResourceMemory) {
			findings = append(findings, types.Finding{
				Resource:  "Pod",
				Namespace: pod.Namespace,
				Name:      pod.Name,
				Severity:  types.SeverityWarning,
				Message:   "container " + c.Name + " is missing a memory limit",
			})
		}
	}
	return findings
}

func hasLimit(limits corev1.ResourceList, name corev1.ResourceName) bool {
	if limits == nil {
		return false
	}
	q, ok := limits[name]
	if !ok {
		return false
	}
	return q.Sign() > 0
}
