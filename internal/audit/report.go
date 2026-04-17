package audit

import (
	"context"
	"sync"

	"github.com/go-crucible/go-crucible/internal/client"
	"github.com/go-crucible/go-crucible/internal/types"
)

// AuditFunc is a function that runs an audit and returns findings. Auditors
// must honor ctx and return promptly when it is cancelled.
type AuditFunc func(ctx context.Context, c client.AuditClient, namespace string) ([]types.Finding, error)

// buildReport aggregates findings into a Report with summary counts.
func buildReport(findings []types.Finding) *types.Report {
	r := &types.Report{Findings: findings}
	r.Summary.Total = len(findings)
	for _, f := range findings {
		switch f.Severity {
		case types.SeverityCritical:
			r.Summary.Critical++
		case types.SeverityWarning:
			r.Summary.Warning++
		case types.SeverityInfo:
			r.Summary.Info++
		}
	}
	return r
}

// ConcurrentAudit runs all provided auditors concurrently and aggregates their
// findings into a single Report.
func ConcurrentAudit(ctx context.Context, auditors []AuditFunc, c client.AuditClient, namespace string) (*types.Report, error) {
	var (
		wg       sync.WaitGroup
		findings []types.Finding
		firstErr error
		errMu    sync.Mutex
	)

	for _, auditor := range auditors {
		wg.Add(1)
		go func(fn AuditFunc) {
			defer wg.Done()
			result, err := fn(ctx, c, namespace)
			if err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMu.Unlock()
				return
			}
			findings = append(findings, result...)
		}(auditor)
	}

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return buildReport(findings), nil
}

// ParallelAudit runs all provided auditors concurrently using a WaitGroup and
// aggregates their findings into a single Report.
func ParallelAudit(ctx context.Context, auditors []AuditFunc, c client.AuditClient, namespace string) (*types.Report, error) {
	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		findings []types.Finding
		firstErr error
	)

	for _, auditor := range auditors {
		go func(fn AuditFunc) {
			wg.Add(1)
			defer wg.Done()

			result, err := fn(ctx, c, namespace)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			mu.Lock()
			findings = append(findings, result...)
			mu.Unlock()
		}(auditor)
	}

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}
	return buildReport(findings), nil
}
