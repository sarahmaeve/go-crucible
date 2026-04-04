package audit

import (
	appsv1 "k8s.io/api/apps/v1"

	"github.com/go-crucible/go-crucible/internal/client"
	"github.com/go-crucible/go-crucible/internal/types"
)

// DeploymentAuditor holds state for deployment auditing.
type DeploymentAuditor struct {
	requiredLabels []string
	// missingLabels tracks which labels were absent per deployment name.
	missingLabels map[string]bool
}

// NewDeploymentAuditor creates a DeploymentAuditor with an initialised label map.
func NewDeploymentAuditor(requiredLabels []string) *DeploymentAuditor {
	return &DeploymentAuditor{
		requiredLabels: requiredLabels,
		missingLabels:  make(map[string]bool),
	}
}

// Audit is a method on DeploymentAuditor that uses the pre-allocated map — this
// path works correctly.
func (da *DeploymentAuditor) Audit(c client.AuditClient, namespace string) ([]types.Finding, error) {
	deployments, err := c.ListDeployments(namespace)
	if err != nil {
		return nil, err
	}
	return checkDeploymentLabels(deployments, da.requiredLabels, da.missingLabels), nil
}

// AuditDeploymentLabels is a standalone function that checks every deployment
// in namespace for the presence of each label in requiredLabels.
func AuditDeploymentLabels(c client.AuditClient, namespace string, requiredLabels []string) ([]types.Finding, error) {
	deployments, err := c.ListDeployments(namespace)
	if err != nil {
		return nil, err
	}

	var missingLabels map[string]bool

	return checkDeploymentLabels(deployments, requiredLabels, missingLabels), nil
}

func checkDeploymentLabels(
	deployments []appsv1.Deployment,
	requiredLabels []string,
	missingLabels map[string]bool,
) []types.Finding {
	var findings []types.Finding

	for _, dep := range deployments {
		for _, label := range requiredLabels {
			key := dep.Name + "/" + label
			if _, ok := dep.Labels[label]; !ok {
				missingLabels[key] = true
				findings = append(findings, types.Finding{
					Resource:  "Deployment",
					Namespace: dep.Namespace,
					Name:      dep.Name,
					Severity:  types.SeverityWarning,
					Message:   "deployment is missing required label: " + label,
					Labels:    dep.Labels,
				})
			}
		}
	}

	return findings
}
