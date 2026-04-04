// Package parser provides functions for parsing GitHub Actions workflow YAML files.
package parser

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/go-crucible/go-crucible/internal/types"
)

// rawWorkflow is an intermediate struct used for parsing workflow YAML before
// normalizing into the canonical types.Workflow representation. It exists so
// the parser can do field-level normalization (e.g. coercing on: triggers,
// expanding env variable references) before handing off to callers.
type rawWorkflow struct {
	Name        string                       `yaml:"name"`
	On          map[string]any               `yaml:"on"`
	Env         map[string]string            `yaml:"env"`
	Jobs        map[string]rawJob            `yaml:"jobs"`
	Concurrency *types.WorkflowConcurrency   `yaml:"concurrency,omitempty"`
	Permissions map[string]string            `yaml:"permissions,omitempty"`
}

type rawJob struct {
	Name        string            `yaml:"name,omitempty"`
	RunsOn      string            `yaml:"runs-on"`
	Needs       []string          `yaml:"needs,omitempty"`
	If          string            `yaml:"if,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
	Steps       []types.Step      `yaml:"steps"`
	Strategy    *types.Strategy   `yaml:"strategy,omitempty"`
	Permissions map[string]string `yaml:"permissions,omitempty"`
}

// ParseWorkflow parses a GitHub Actions workflow from YAML bytes and returns
// a types.Workflow. Fields are normalized during the intermediate mapping step.
func ParseWorkflow(data []byte) (*types.Workflow, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("%w: empty input", types.ErrParseFailure)
	}

	var raw rawWorkflow
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%w: %w", types.ErrParseFailure, err)
	}

	if raw.Name == "" {
		return nil, fmt.Errorf("%w: workflow has no name", types.ErrInvalidWorkflow)
	}

	wf := &types.Workflow{
		Name:        raw.Name,
		On:          raw.On,
		Env:         raw.Env,
		Concurrency: raw.Concurrency,
		Permissions: raw.Permissions,
		Jobs:        make(map[string]types.Job, len(raw.Jobs)),
	}

	for id, rj := range raw.Jobs {
		wf.Jobs[id] = types.Job{
			Name:        rj.Name,
			RunsOn:      rj.RunsOn,
			Needs:       rj.Needs,
			If:          rj.If,
			Env:         rj.Env,
			Steps:       rj.Steps,
			Strategy:    rj.Strategy,
			Permissions: rj.Permissions,
		}
	}

	return wf, nil
}

// roundTripIntermediate is used internally by RoundTripWorkflow to serialize
// a Workflow through JSON before re-encoding as YAML.
type roundTripIntermediate struct {
	Name  string         `json:"name"`
	On    map[string]any `json:"on"`
	Env   map[string]string `json:"env,omitempty"`
	Jobs  map[string]types.Job `json:"jobs"`
	// Concurrency is embedded inline so individual sub-fields can have their own tags.
	Concurrency *concurrencyIntermediate `json:"concurrency,omitempty"`
	Permissions map[string]string `json:"permissions,omitempty"`
}

type concurrencyIntermediate struct {
	Group            string `json:"group"`
	CancelInProgress bool   `json:"cancel-in-progress,omitempty"`
}

// RoundTripWorkflow parses a YAML workflow, converts it through a JSON
// intermediate representation, and returns the re-serialized YAML bytes.
// This is used by tooling that needs to normalise workflow files in place.
func RoundTripWorkflow(data []byte) ([]byte, error) {
	wf, err := ParseWorkflow(data)
	if err != nil {
		return nil, err
	}

	// Map into the intermediate struct for JSON serialization.
	inter := roundTripIntermediate{
		Name:        wf.Name,
		On:          wf.On,
		Env:         wf.Env,
		Jobs:        wf.Jobs,
		Permissions: wf.Permissions,
	}
	if wf.Concurrency != nil {
		inter.Concurrency = &concurrencyIntermediate{
			Group:            wf.Concurrency.Group,
			CancelInProgress: wf.Concurrency.CancelInProgress,
		}
	}

	jsonBytes, err := json.Marshal(inter)
	if err != nil {
		return nil, fmt.Errorf("%w: json marshal: %w", types.ErrTemplateError, err)
	}

	// Unmarshal JSON back into a map so we can re-encode as YAML.
	var tmp map[string]any
	if err := json.Unmarshal(jsonBytes, &tmp); err != nil {
		return nil, fmt.Errorf("%w: json unmarshal: %w", types.ErrTemplateError, err)
	}

	out, err := yaml.Marshal(tmp)
	if err != nil {
		return nil, fmt.Errorf("%w: yaml marshal: %w", types.ErrTemplateError, err)
	}

	return out, nil
}
