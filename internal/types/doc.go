// Package types holds the shared data model used across go-crucible's three
// applications.
//
// The types are split by application to keep each domain's vocabulary
// self-contained:
//
//   - pipeline.go — the metric-ingestion and alerting daemon (cmd/pipeline).
//     Defines [Metric], [Sample], [Alert], [AlertRule], and [AlertState].
//   - kubepatrol.go — the Kubernetes auditor CLI (cmd/kube-patrol). Defines
//     [Finding], [Report], [Summary], and [Severity].
//   - ghforge.go — the GitHub Actions workflow tool (cmd/gh-forge). Defines
//     [Workflow], [Job], [Step], [Strategy], [MatrixCombination],
//     [LintFinding], and [ValidationError].
//   - errors.go — sentinel error values shared across the three apps.
//
// Types here are intentionally simple data holders. Behaviour lives in the
// per-domain packages under internal/ that operate on these types.
package types
