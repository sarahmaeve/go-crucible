package generate_test

import (
	"slices"
	"testing"

	"github.com/go-crucible/go-crucible/internal/generate"
)

func TestBaseTemplatePinsCurrentGo(t *testing.T) {
	wf, err := (generate.BaseTemplate{WorkflowName: "CI"}).Generate()
	if err != nil {
		t.Fatalf("Generate() returned unexpected error: %v", err)
	}

	steps := wf.Jobs["test"].Steps
	if len(steps) < 2 {
		t.Fatalf("test job has %d steps; want at least checkout and Go setup", len(steps))
	}
	if got, want := steps[1].Uses, "actions/setup-go@v5"; got != want {
		t.Errorf("Go setup action = %q; want %q", got, want)
	}
	if got, want := steps[1].With["go-version"], "1.27.0"; got != want {
		t.Errorf("generated Go version = %q; want %q", got, want)
	}
}

func TestAdvancedTemplateDefaultsToCurrentGo(t *testing.T) {
	wf, err := (generate.AdvancedTemplate{
		BaseTemplate: generate.BaseTemplate{WorkflowName: "Advanced CI"},
	}).Generate()
	if err != nil {
		t.Fatalf("Generate() returned unexpected error: %v", err)
	}

	if got, want := wf.Jobs["test"].Strategy.Matrix["go"], []string{"1.27.0"}; !slices.Equal(got, want) {
		t.Errorf("default matrix.go = %v; want %v", got, want)
	}
}

func TestDefaultAdvancedTemplatePinsCurrentGo(t *testing.T) {
	tmpl, err := generate.DefaultRegistry.New("advanced")
	if err != nil {
		t.Fatalf("DefaultRegistry.New() returned unexpected error: %v", err)
	}
	wf, err := tmpl.Generate()
	if err != nil {
		t.Fatalf("Generate() returned unexpected error: %v", err)
	}

	if got, want := wf.Jobs["test"].Strategy.Matrix["go"], []string{"1.27.0"}; !slices.Equal(got, want) {
		t.Errorf("registry matrix.go = %v; want %v", got, want)
	}
}

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
		if got, want := matrix["os"], []string{"ubuntu-latest", "macos-latest"}; !slices.Equal(got, want) {
			t.Errorf("matrix.os = %v; want %v", got, want)
		}
		if got, want := matrix["go"], []string{"1.21", "1.22"}; !slices.Equal(got, want) {
			t.Errorf("matrix.go = %v; want %v", got, want)
		}
	}

	// --- Assert concurrency block is present ---
	if wf.Concurrency == nil {
		t.Errorf("workflow Concurrency is nil; expected concurrency group to be set")
	} else if got, want := wf.Concurrency.Group, "ci-${{ github.ref }}"; got != want {
		t.Errorf("Concurrency.Group = %q; want %q", got, want)
	}

	if got, want := wf.Name, "Advanced CI"; got != want {
		t.Errorf("workflow name = %q; want %q", got, want)
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
