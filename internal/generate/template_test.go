package generate_test

import (
	"testing"

	"github.com/go-crucible/go-crucible/internal/generate"
)

// TestExercise11_TemplateTrap verifies that BuildAdvancedTemplate returns a
// Template whose Generate method produces a workflow with a matrix strategy
// and concurrency settings matching the provided configuration.
func TestExercise11_TemplateTrap(t *testing.T) {
	tmpl := generate.BuildAdvancedTemplate(
		"Advanced CI",
		[]string{"ubuntu-latest", "macos-latest"},
		[]string{"1.21", "1.22"},
		"ci-${{ github.ref }}",
	)

	wf, err := tmpl.Generate()
	if err != nil {
		t.Fatalf("Generate() returned unexpected error: %v", err)
	}

	// --- Assert matrix strategy is present ---
	testJob, ok := wf.Jobs["test"]
	if !ok {
		t.Fatalf("generated workflow has no 'test' job; jobs: %v", wf.Jobs)
	}

	if testJob.Strategy == nil {
		t.Errorf("test job Strategy is nil; expected a matrix strategy with os and go axes")
	} else {
		matrix := testJob.Strategy.Matrix
		if len(matrix["os"]) < 2 {
			t.Errorf("matrix.os has %d entries; want at least 2 (ubuntu-latest, macos-latest)", len(matrix["os"]))
		}
		if len(matrix["go"]) < 2 {
			t.Errorf("matrix.go has %d entries; want at least 2 (1.21, 1.22)", len(matrix["go"]))
		}
	}

	// --- Assert concurrency block is present ---
	if wf.Concurrency == nil {
		t.Errorf("workflow Concurrency is nil; expected concurrency group to be set")
	} else if wf.Concurrency.Group == "" {
		t.Errorf("Concurrency.Group is empty; expected 'ci-${{ github.ref }}'")
	}

	// --- Assert the runner uses the matrix variable ---
	if testJob.RunsOn != "${{ matrix.os }}" {
		t.Errorf("RunsOn = %q; want \"${{ matrix.os }}\" (set by AdvancedTemplate.Generate)", testJob.RunsOn)
	}

	// --- Extra diagnostic ---
	if t.Failed() {
		t.Logf("Template type returned by BuildAdvancedTemplate: %T", tmpl)
		t.Logf("Generated workflow: name=%q jobs=%d concurrency=%v",
			wf.Name, len(wf.Jobs), wf.Concurrency)
		if testJob.Strategy != nil {
			t.Logf("  strategy.matrix=%v", testJob.Strategy.Matrix)
		} else {
			t.Log("  strategy: nil")
		}
	}
}
