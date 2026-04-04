package lint_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"

	"github.com/go-crucible/go-crucible/internal/lint"
)

// minimalWorkflowYAML is a valid, named workflow with one job and one step.
// It uses a short action version tag so pin-actions-version fires but the
// file is otherwise well-formed.
const minimalWorkflowYAML = `name: Lint Test Workflow
on:
  push:
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
`

// TestExercise16_LeakingLinter creates 550 YAML files in a temp directory,
// lowers the open-file-descriptor limit to 256, runs LintWorkflows, and
// asserts no "too many open files" error is returned. The test verifies that
// LintWorkflows closes file descriptors promptly during iteration rather than
// accumulating them all until the function returns.
func TestExercise16_LeakingLinter(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fd-limit test not applicable on Windows")
	}

	// --- Lower the open-file descriptor limit ---
	// We set both soft and hard limits to a modest value so the test is
	// self-contained. Any value comfortably below 550 (our file count) but
	// comfortably above what the test process itself needs (stdin/stdout/stderr
	// + a handful of runtime fds) works.
	const fdLimit = 256
	const fileCount = 550

	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		t.Fatalf("Getrlimit: %v", err)
	}
	original := rLimit

	// Only lower if the current limit is higher than what we want to impose.
	if rLimit.Cur > fdLimit {
		rLimit.Cur = fdLimit
		// Leave Hard unchanged; we only need to lower the soft limit.
		if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
			t.Skipf("cannot set RLIMIT_NOFILE (may need elevated privileges): %v", err)
		}
		t.Cleanup(func() {
			// Restore original limits so other tests are not affected.
			_ = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &original)
		})
	}

	// --- Populate temp directory with YAML files ---
	dir := t.TempDir()
	for i := 0; i < fileCount; i++ {
		path := filepath.Join(dir, fmt.Sprintf("workflow_%04d.yml", i))
		if err := os.WriteFile(path, []byte(minimalWorkflowYAML), 0o644); err != nil {
			t.Fatalf("writing test file: %v", err)
		}
	}

	// --- Run the linter ---
	_, err := lint.LintWorkflows(dir)
	if err != nil {
		t.Errorf("LintWorkflows returned an error with %d files and fdLimit=%d: %v",
			fileCount, fdLimit, err)
	}
}

// TestLintWorkflows_BasicFindings is a sanity-check test that verifies the
// linter actually produces findings for known-bad input. It is NOT an exercise
// bug test — it just confirms the linter machinery works.
func TestLintWorkflows_BasicFindings(t *testing.T) {
	dir := t.TempDir()

	// A workflow without a name should trigger workflow-name-required.
	noNameYAML := `on:
  push:
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
`
	if err := os.WriteFile(filepath.Join(dir, "no-name.yml"), []byte(noNameYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	findings, err := lint.LintWorkflows(dir)
	if err != nil {
		t.Fatalf("LintWorkflows returned unexpected error: %v", err)
	}

	found := false
	for _, f := range findings {
		if f.Rule == "workflow-name-required" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'workflow-name-required' finding for nameless workflow, got findings: %v", findings)
	}
}

// TestLintWorkflows_EmptyDir verifies the linter handles an empty directory gracefully.
func TestLintWorkflows_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	findings, err := lint.LintWorkflows(dir)
	if err != nil {
		t.Fatalf("LintWorkflows on empty dir returned error: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected no findings for empty dir, got %d", len(findings))
	}
}
