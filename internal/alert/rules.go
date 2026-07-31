package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/go-crucible/go-crucible/internal/types"
)

// FetchRules downloads and combines rule documents from URLs using client,
// which must be non-nil. Each document must contain the same JSON array
// accepted by [LoadRules].
func FetchRules(ctx context.Context, client *http.Client, urls []string) ([]types.AlertRule, error) {
	var all []types.AlertRule
	for _, url := range urls {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("alert: creating rule request for %q: %w", url, err)
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("alert: fetching rules from %q: %w", url, err)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("alert: fetching rules from %q: status %s", url, resp.Status)
		}
		rules, err := LoadRules(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("alert: fetching rules from %q: %w", url, err)
		}
		all = append(all, rules...)
	}
	return all, nil
}

// LoadRules decodes a JSON array of AlertRule values from r.
func LoadRules(r io.Reader) ([]types.AlertRule, error) {
	var rules []types.AlertRule
	dec := json.NewDecoder(r)
	if err := dec.Decode(&rules); err != nil {
		return nil, fmt.Errorf("alert: failed to decode rules: %w", err)
	}
	return rules, nil
}

// MatchingRules returns the subset of rules whose MetricName equals metricName.
func MatchingRules(rules []types.AlertRule, metricName string) []types.AlertRule {
	var out []types.AlertRule
	for _, r := range rules {
		if r.MetricName == metricName {
			out = append(out, r)
		}
	}
	return out
}
