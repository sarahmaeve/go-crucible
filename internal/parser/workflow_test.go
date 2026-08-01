package parser_test

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

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

	// The source YAML defines both triggers, so both must survive parsing.
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

	// The source YAML also defines two top-level environment variables.
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

// TestExercise15_ConfigSurprise verifies all three presence states for the
// optional cancel-in-progress setting survive a workflow round trip.
func TestExercise15_ConfigSurprise(t *testing.T) {
	for _, tc := range []struct {
		name      string
		setting   string
		wantField bool
		wantValue bool
	}{
		{name: "absent"},
		{name: "explicit false", setting: "  cancel-in-progress: false\n", wantField: true},
		{name: "explicit true", setting: "  cancel-in-progress: true\n", wantField: true, wantValue: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			workflowYAML := `
name: Deploy

on:
  push:
    branches: [main]

concurrency:
  group: deploy-prod
` + tc.setting + `
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

			var decoded struct {
				Concurrency map[string]any `yaml:"concurrency"`
			}
			if err := yaml.Unmarshal(out, &decoded); err != nil {
				t.Fatalf("round-tripped output is not valid YAML: %v\n%s", err, out)
			}
			value, ok := decoded.Concurrency["cancel-in-progress"]
			if ok != tc.wantField {
				t.Fatalf("cancel-in-progress presence = %t, want %t\n%s", ok, tc.wantField, out)
			}
			if ok {
				got, isBool := value.(bool)
				if !isBool || got != tc.wantValue {
					t.Errorf("cancel-in-progress = %#v, want %t\n%s", value, tc.wantValue, out)
				}
			}
		})
	}
}
