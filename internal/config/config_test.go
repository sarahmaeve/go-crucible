package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTemp writes content to a temporary file and returns its path. The file
// is removed automatically when the test finishes.
func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.yaml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close temp file: %v", err)
	}
	return f.Name()
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Severity != "info" {
		t.Errorf("default severity = %q, want %q", cfg.Severity, "info")
	}
	if len(cfg.Namespaces) != 0 {
		t.Errorf("default namespaces should be empty, got %v", cfg.Namespaces)
	}
	if len(cfg.RequiredLabels) != 0 {
		t.Errorf("default required_labels should be empty, got %v", cfg.RequiredLabels)
	}
	if len(cfg.SkipAudits) != 0 {
		t.Errorf("default skip_audits should be empty, got %v", cfg.SkipAudits)
	}
}

func TestLoad_FullConfig(t *testing.T) {
	yaml := `
namespaces:
  - default
  - kube-system
required_labels:
  - app
  - team
skip_audits:
  - resource-limits
severity: warning
`
	path := writeTemp(t, yaml)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	wantNamespaces := []string{"default", "kube-system"}
	if len(cfg.Namespaces) != len(wantNamespaces) {
		t.Fatalf("namespaces len = %d, want %d", len(cfg.Namespaces), len(wantNamespaces))
	}
	for i, ns := range wantNamespaces {
		if cfg.Namespaces[i] != ns {
			t.Errorf("namespaces[%d] = %q, want %q", i, cfg.Namespaces[i], ns)
		}
	}

	wantLabels := []string{"app", "team"}
	if len(cfg.RequiredLabels) != len(wantLabels) {
		t.Fatalf("required_labels len = %d, want %d", len(cfg.RequiredLabels), len(wantLabels))
	}
	for i, lbl := range wantLabels {
		if cfg.RequiredLabels[i] != lbl {
			t.Errorf("required_labels[%d] = %q, want %q", i, cfg.RequiredLabels[i], lbl)
		}
	}

	if len(cfg.SkipAudits) != 1 || cfg.SkipAudits[0] != "resource-limits" {
		t.Errorf("skip_audits = %v, want [resource-limits]", cfg.SkipAudits)
	}

	if cfg.Severity != "warning" {
		t.Errorf("severity = %q, want %q", cfg.Severity, "warning")
	}
}

func TestLoad_DefaultsApplied(t *testing.T) {
	// An empty config file should still produce valid defaults.
	path := writeTemp(t, "{}\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Severity != "info" {
		t.Errorf("severity with empty file = %q, want %q", cfg.Severity, "info")
	}
}

func TestLoad_InvalidSeverity(t *testing.T) {
	yaml := "severity: loud\n"
	path := writeTemp(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() expected error for invalid severity, got nil")
	}
}

func TestLoad_UnknownField(t *testing.T) {
	yaml := "not_a_real_field: oops\n"
	path := writeTemp(t, yaml)
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() expected error for unknown field, got nil")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.yaml")
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() expected error for missing file, got nil")
	}
}

func TestLoad_MalformedYAML(t *testing.T) {
	path := writeTemp(t, "namespaces: [\n") // unclosed bracket
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() expected error for malformed YAML, got nil")
	}
}

func TestLoad_AllSeverityLevels(t *testing.T) {
	for _, sev := range []string{"info", "warning", "critical"} {
		t.Run(sev, func(t *testing.T) {
			yaml := "severity: " + sev + "\n"
			path := writeTemp(t, yaml)
			cfg, err := Load(path)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if cfg.Severity != sev {
				t.Errorf("severity = %q, want %q", cfg.Severity, sev)
			}
		})
	}
}
