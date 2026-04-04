package parser_test

import (
	"strings"
	"testing"

	"github.com/go-crucible/go-crucible/internal/parser"
)

// TestExercise04_MissingWorkflow parses a workflow YAML that has all fields
// populated and asserts that every field in the returned Workflow is non-zero.
func TestExercise04_MissingWorkflow(t *testing.T) {
	const workflowYAML = `
name: My CI Workflow

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

env:
  GO_VERSION: "1.22"
  GOFLAGS: "-mod=vendor"

concurrency:
  group: ci-${{ github.ref }}
  cancel-in-progress: true

permissions:
  contents: read

jobs:
  test:
    runs-on: ubuntu-latest
    env:
      CGO_ENABLED: "0"
    steps:
      - name: Checkout
        uses: actions/checkout@v4
      - name: Test
        run: go test ./...
`

	wf, err := parser.ParseWorkflow([]byte(workflowYAML))
	if err != nil {
		t.Fatalf("ParseWorkflow returned unexpected error: %v", err)
	}

	// Name should parse correctly.
	if wf.Name == "" {
		t.Errorf("Name is empty; expected 'My CI Workflow'")
	}

	// Jobs should parse correctly.
	if len(wf.Jobs) == 0 {
		t.Errorf("Jobs is empty; expected at least one job")
	}

	// The 'on' triggers map must be non-nil — the source YAML has push and
	// pull_request triggers. FAILS because rawWorkflow.on is unexported.
	if wf.On == nil {
		t.Errorf("On (triggers) is nil; YAML contains 'on: push/pull_request' but the intermediate struct field is unexported and the YAML decoder skips it")
	} else {
		if _, ok := wf.On["push"]; !ok {
			t.Errorf("On map is missing 'push' key; got keys: %v", mapKeys(wf.On))
		}
		if _, ok := wf.On["pull_request"]; !ok {
			t.Errorf("On map is missing 'pull_request' key; got keys: %v", mapKeys(wf.On))
		}
	}

	// The top-level env map must be non-nil. FAILS because rawWorkflow.env is unexported.
	if wf.Env == nil {
		t.Errorf("Env is nil; YAML contains top-level env vars but the intermediate struct field is unexported and the YAML decoder skips it")
	} else {
		if wf.Env["GO_VERSION"] == "" {
			t.Errorf("Env[GO_VERSION] is empty; expected '1.22'")
		}
		if wf.Env["GOFLAGS"] == "" {
			t.Errorf("Env[GOFLAGS] is empty; expected '-mod=vendor'")
		}
	}

	// Concurrency should be present.
	if wf.Concurrency == nil {
		t.Errorf("Concurrency is nil; YAML defines a concurrency block")
	} else {
		if !strings.Contains(wf.Concurrency.Group, "ci-") {
			t.Errorf("Concurrency.Group = %q; want something containing 'ci-'", wf.Concurrency.Group)
		}
	}

	// Permissions should be present.
	if len(wf.Permissions) == 0 {
		t.Errorf("Permissions is empty; YAML defines permissions")
	}
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// TestExercise15_ConfigSurprise round-trips a workflow YAML that has
// cancel-in-progress: false and asserts the value is preserved.
//
// The test FAILS because concurrencyIntermediate.CancelInProgress has
// json:"cancel-in-progress,omitempty" — a false bool is its zero value and
// omitempty silently drops it, changing the semantic meaning of the workflow.
func TestExercise15_ConfigSurprise(t *testing.T) {
	// This workflow deliberately sets cancel-in-progress: false to ensure
	// deployments are serialized rather than cancelled.
	const workflowYAML = `
name: Deploy

on:
  push:
    branches: [main]

concurrency:
  group: deploy-prod
  cancel-in-progress: false

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Deploy
        run: ./deploy.sh
`

	out, err := parser.RoundTripWorkflow([]byte(workflowYAML))
	if err != nil {
		t.Fatalf("RoundTripWorkflow returned unexpected error: %v", err)
	}

	outStr := string(out)

	// The round-tripped YAML must still contain the cancel-in-progress key.
	// FAILS because JSON omitempty drops the false bool during marshaling.
	if !strings.Contains(outStr, "cancel-in-progress") {
		t.Errorf("Round-tripped YAML is missing 'cancel-in-progress' key entirely.\n"+
			"This means a deliberate 'false' value was silently dropped by omitempty.\n"+
			"Output:\n%s", outStr)
	} else if strings.Contains(outStr, "cancel-in-progress: false") || strings.Contains(outStr, "cancel-in-progress: \"false\"") {
		// Value is present and correct.
		t.Logf("cancel-in-progress: false correctly preserved in output")
	} else {
		// Key is present but value might be wrong.
		t.Logf("Output contains 'cancel-in-progress' but value may be incorrect:\n%s", outStr)
	}
}
