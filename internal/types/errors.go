package types

import "errors"

// Sentinel errors used across the applications.
var (
	// kube-patrol errors
	ErrClientNotReady = errors.New("kubernetes client is not ready")
	ErrAuditFailed    = errors.New("audit operation failed")

	// pipeline errors
	ErrThresholdExceeded = errors.New("metric threshold exceeded")
	ErrSourceDrained     = errors.New("metric source has been drained")
	ErrPipelineShutdown  = errors.New("pipeline is shutting down")

	// gh-forge errors
	ErrInvalidWorkflow = errors.New("invalid workflow file")
	ErrParseFailure    = errors.New("failed to parse workflow")
	ErrTemplateError   = errors.New("template rendering failed")
)
