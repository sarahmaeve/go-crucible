package types

import "errors"

// Sentinel errors used across the applications. Callers should prefer
// [errors.Is] over string comparison when checking for these.
var (
	// ErrClientNotReady is returned when a call is made against an
	// uninitialised or disconnected Kubernetes client. (kube-patrol)
	ErrClientNotReady = errors.New("kubernetes client is not ready")

	// ErrAuditFailed indicates an audit could not complete. The wrapped error
	// (via %w) carries the underlying cause. (kube-patrol)
	ErrAuditFailed = errors.New("audit operation failed")

	// ErrThresholdExceeded signals that a metric value crossed its alert rule
	// threshold. Alert evaluators wrap this with %w so that callers can use
	// [errors.Is] to distinguish threshold crossings from other errors.
	// (pipeline)
	ErrThresholdExceeded = errors.New("metric threshold exceeded")

	// ErrSourceDrained is returned by a [MetricSource] once no further
	// metrics are available. It is a clean end-of-stream signal, not a
	// failure. (pipeline)
	ErrSourceDrained = errors.New("metric source has been drained")

	// ErrPipelineShutdown is returned by pipeline operations that are
	// terminating because the pipeline's context was cancelled. (pipeline)
	ErrPipelineShutdown = errors.New("pipeline is shutting down")

	// ErrDuplicate is returned by [CacheStore] implementations when a key has
	// already been written. Deduplicating callers treat this as an idempotent
	// success. Implementations should wrap this sentinel with %w so that
	// callers can use [errors.Is] to recognise it regardless of the
	// surrounding message. (pipeline)
	ErrDuplicate = errors.New("duplicate write rejected")

	// ErrInvalidWorkflow indicates a workflow failed semantic validation
	// (required fields missing, disallowed values, etc.). (gh-forge)
	ErrInvalidWorkflow = errors.New("invalid workflow file")

	// ErrParseFailure indicates the YAML/JSON for a workflow could not be
	// decoded into the workflow data model. (gh-forge)
	ErrParseFailure = errors.New("failed to parse workflow")

	// ErrTemplateError indicates a generator template failed to produce a
	// valid workflow. (gh-forge)
	ErrTemplateError = errors.New("template rendering failed")
)
