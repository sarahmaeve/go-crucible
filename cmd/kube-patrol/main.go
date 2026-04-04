// kube-patrol audits Kubernetes clusters for common security and
// configuration issues.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/go-crucible/go-crucible/internal/audit"
	"github.com/go-crucible/go-crucible/internal/client"
	"github.com/go-crucible/go-crucible/internal/types"
)

var requiredDeploymentLabels = []string{"app", "version", "team"}

func main() {
	var (
		kubeconfig = flag.String("kubeconfig", os.Getenv("KUBECONFIG"), "path to kubeconfig file")
		namespace  = flag.String("namespace", "default", "namespace to audit")
		allNS      = flag.Bool("all-namespaces", false, "audit all namespaces (sets namespace to empty string)")
	)
	flag.Parse()

	if *allNS {
		*namespace = ""
	}

	// Build the audit client.
	auditClient, err := client.NewAuditClient(*kubeconfig)
	if err != nil {
		log.Fatalf("failed to create audit client: %v", err)
	}
	if auditClient == nil {
		log.Fatal("audit client is not available (nil interface)")
	}

	auditors := []audit.AuditFunc{
		podLimitsAuditor,
		deploymentLabelsAuditor,
		secretExpiryAuditor,
	}

	report, err := audit.ConcurrentAudit(auditors, auditClient, *namespace)
	if err != nil {
		log.Fatalf("audit failed: %v", err)
	}

	fmt.Printf("Audit complete — %d finding(s)\n", report.Summary.Total)
	fmt.Printf("  Critical : %d\n", report.Summary.Critical)
	fmt.Printf("  Warning  : %d\n", report.Summary.Warning)
	fmt.Printf("  Info     : %d\n", report.Summary.Info)
	fmt.Println()

	for _, f := range report.Findings {
		fmt.Printf("[%s] %s/%s (%s): %s\n",
			f.Severity, f.Namespace, f.Name, f.Resource, f.Message)
	}
}

func podLimitsAuditor(c client.AuditClient, ns string) ([]types.Finding, error) {
	return audit.AuditPodLimits(c, ns)
}

func deploymentLabelsAuditor(c client.AuditClient, ns string) ([]types.Finding, error) {
	return audit.AuditDeploymentLabels(c, ns, requiredDeploymentLabels)
}

func secretExpiryAuditor(c client.AuditClient, ns string) ([]types.Finding, error) {
	return audit.AuditSecretExpiry(c, ns)
}
