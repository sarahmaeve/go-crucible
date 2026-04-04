package alert

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/go-crucible/go-crucible/internal/types"
)

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
