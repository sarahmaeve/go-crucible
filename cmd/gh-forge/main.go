// Command gh-forge provides GitHub Actions workflow tooling: parsing,
// validation, template generation, and linting.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/go-crucible/go-crucible/internal/generate"
	"github.com/go-crucible/go-crucible/internal/lint"
	"github.com/go-crucible/go-crucible/internal/parser"
	"github.com/go-crucible/go-crucible/internal/validate"
)

const usage = `gh-forge — GitHub Actions workflow tooling

Usage:
  gh-forge <command> [args]

Commands:
  validate <file>       Validate a workflow YAML file
  generate <template>   Generate a workflow from a named template
  lint <dir>            Lint all workflow YAML files in a directory
  roundtrip <file>      Parse and re-serialize a workflow YAML file

Available templates: basic, advanced
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "validate":
		err = runValidate(os.Args[2:])
	case "generate":
		err = runGenerate(os.Args[2:])
	case "lint":
		err = runLint(os.Args[2:])
	case "roundtrip":
		err = runRoundtrip(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %q\n\n%s", os.Args[1], usage)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runValidate(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("validate requires a workflow file argument")
	}
	path := args[0]
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %q: %w", path, err)
	}

	wf, err := parser.ParseWorkflow(data)
	if err != nil {
		return fmt.Errorf("parsing workflow: %w", err)
	}

	errs, err := validate.ValidateWorkflow(wf)
	if err != nil {
		return fmt.Errorf("validation: %w", err)
	}

	if len(errs) == 0 {
		fmt.Printf("OK: %q is valid\n", path)
		return nil
	}

	fmt.Printf("FAIL: %d validation error(s) in %q:\n", len(errs), path)
	for _, e := range errs {
		fmt.Printf("  • [%s] %s\n", e.Field, e.Message)
	}
	return fmt.Errorf("%d validation error(s)", len(errs))
}

func runGenerate(args []string) error {
	templateName := "basic"
	if len(args) > 0 {
		templateName = args[0]
	}

	tmpl, err := generate.DefaultRegistry.New(templateName)
	if err != nil {
		return fmt.Errorf("template %q: %w", templateName, err)
	}

	wf, err := tmpl.Generate()
	if err != nil {
		return fmt.Errorf("generating workflow: %w", err)
	}

	out, err := yaml.Marshal(wf)
	if err != nil {
		return fmt.Errorf("marshaling workflow: %w", err)
	}

	fmt.Print(string(out))
	return nil
}

func runLint(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("lint requires a directory argument")
	}
	dir := args[0]

	findings, err := lint.LintWorkflows(dir)
	if err != nil {
		return fmt.Errorf("linting: %w", err)
	}

	if len(findings) == 0 {
		fmt.Printf("OK: no issues found in %q\n", dir)
		return nil
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(findings); err != nil {
		return fmt.Errorf("encoding findings: %w", err)
	}

	counts := lint.CountBySeverity(findings)
	fmt.Fprintf(os.Stderr, "\n%d finding(s): errors=%d warnings=%d info=%d\n",
		len(findings), counts["error"], counts["warning"], counts["info"])

	if counts["error"] > 0 {
		return fmt.Errorf("%d error-level finding(s)", counts["error"])
	}
	return nil
}

func runRoundtrip(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("roundtrip requires a workflow file argument")
	}
	path := args[0]

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %q: %w", path, err)
	}

	out, err := parser.RoundTripWorkflow(data)
	if err != nil {
		return fmt.Errorf("round-trip: %w", err)
	}

	fmt.Print(string(out))
	return nil
}
