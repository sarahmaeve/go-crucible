// kube-patrol audits Kubernetes clusters for common security and
// configuration issues.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-crucible/go-crucible/internal/audit"
	"github.com/go-crucible/go-crucible/internal/client"
	"github.com/go-crucible/go-crucible/internal/types"
)

var requiredDeploymentLabels = []string{"app", "version", "team"}

// fatalf logs msg at error level with the given key-value pairs and exits 1.
// It replaces log.Fatalf now that the CLI uses log/slog.
func fatalf(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}

func main() {
	// Install a single signal-aware context for the whole audit run so Ctrl-C
	// or SIGTERM cancels in-flight Kubernetes API calls promptly.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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
		fatalf("failed to create audit client", "err", err)
	}
	if auditClient == nil {
		fatalf("audit client is not available (nil interface)")
	}

	auditors := []audit.AuditFunc{
		podLimitsAuditor,
		deploymentLabelsAuditor,
		secretExpiryAuditor,
	}

	report, err := audit.ConcurrentAudit(ctx, auditors, auditClient, *namespace)
	if err != nil {
		fatalf("audit failed", "err", err)
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

func podLimitsAuditor(ctx context.Context, c client.AuditClient, ns string) ([]types.Finding, error) {
	return audit.AuditPodLimits(ctx, c, ns)
}

func deploymentLabelsAuditor(ctx context.Context, c client.AuditClient, ns string) ([]types.Finding, error) {
	return audit.AuditDeploymentLabels(ctx, c, ns, requiredDeploymentLabels)
}

func secretExpiryAuditor(ctx context.Context, c client.AuditClient, ns string) ([]types.Finding, error) {
	return audit.AuditSecretExpiry(ctx, c, ns)
}
