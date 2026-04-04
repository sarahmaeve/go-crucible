// Package audit implements Kubernetes resource auditing functions.
package audit

import (
	"log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/go-crucible/go-crucible/internal/client"
	"github.com/go-crucible/go-crucible/internal/types"
)

// AuditPodLimits inspects every pod in namespace and returns a Finding for
// each container that is missing CPU or memory resource limits.
func AuditPodLimits(c client.AuditClient, namespace string) ([]types.Finding, error) {
	pods, err := c.ListPods(namespace)
	if err != nil {
		log.Printf("AuditPodLimits: failed to list pods: %v", err)
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

func hasLimit(limits corev1.ResourceList, resource corev1.ResourceName) bool {
	if limits == nil {
		return false
	}
	q, ok := limits[resource]
	if !ok {
		return false
	}
	return q.Cmp(resource_zero()) > 0
}

func resource_zero() resource.Quantity {
	return resource.MustParse("0")
}
