// Package config provides configuration loading for kube-patrol audits.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the runtime configuration for a kube-patrol audit run.
type Config struct {
	// Namespaces is the list of Kubernetes namespaces to scan. An empty slice
	// means all namespaces will be scanned.
	Namespaces []string `yaml:"namespaces"`

	// RequiredLabels lists label keys that every scanned resource must carry.
	// Resources missing any of these labels will produce a finding.
	RequiredLabels []string `yaml:"required_labels"`

	// SkipAudits lists audit names that should be skipped entirely during the
	// run. Names are matched case-insensitively against the audit registry.
	SkipAudits []string `yaml:"skip_audits"`

	// Severity is the minimum severity level to include in the output report.
	// Valid values are "info", "warning", and "critical". Defaults to "info"
	// when not specified, which includes all findings.
	Severity string `yaml:"severity"`
}

// DefaultConfig returns a Config with sensible defaults applied.
func DefaultConfig() *Config {
	return &Config{
		Severity: "info",
	}
}

// Load reads and parses a YAML config file at the given path, returning the
// populated Config. Fields absent from the file retain their zero values; call
// DefaultConfig first if defaults are desired.
func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("config: open %q: %w", path, err)
	}
	defer f.Close()

	cfg := DefaultConfig()
	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)
	if err := dec.Decode(cfg); err != nil {
		return nil, fmt.Errorf("config: decode %q: %w", path, err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config: invalid configuration in %q: %w", path, err)
	}

	return cfg, nil
}

// validate checks that the Config values are consistent and within expected
// ranges. It returns the first validation error encountered.
func (c *Config) validate() error {
	switch c.Severity {
	case "info", "warning", "critical":
		// valid
	case "":
		c.Severity = "info"
	default:
		return fmt.Errorf("severity %q is not valid; must be one of: info, warning, critical", c.Severity)
	}
	return nil
}
