package types

// Workflow represents a GitHub Actions workflow file.
type Workflow struct {
	Name        string                `yaml:"name" json:"name"`
	On          map[string]any        `yaml:"on" json:"on"`
	Env         map[string]string     `yaml:"env,omitempty" json:"env,omitempty"`
	Jobs        map[string]Job        `yaml:"jobs" json:"jobs"`
	Concurrency *WorkflowConcurrency  `yaml:"concurrency,omitempty" json:"concurrency,omitempty"`
	Permissions map[string]string     `yaml:"permissions,omitempty" json:"permissions,omitempty"`
}

// WorkflowConcurrency controls concurrent workflow runs.
type WorkflowConcurrency struct {
	Group            string `yaml:"group" json:"group"`
	CancelInProgress bool   `yaml:"cancel-in-progress" json:"cancel-in-progress"`
}

// Job represents a single job in a workflow.
type Job struct {
	Name        string            `yaml:"name,omitempty" json:"name,omitempty"`
	RunsOn      string            `yaml:"runs-on" json:"runs-on"`
	Needs       []string          `yaml:"needs,omitempty" json:"needs,omitempty"`
	If          string            `yaml:"if,omitempty" json:"if,omitempty"`
	Env         map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	Steps       []Step            `yaml:"steps" json:"steps"`
	Strategy    *Strategy         `yaml:"strategy,omitempty" json:"strategy,omitempty"`
	Permissions map[string]string `yaml:"permissions,omitempty" json:"permissions,omitempty"`
}

// Step represents a single step within a job.
type Step struct {
	Name  string            `yaml:"name,omitempty" json:"name,omitempty"`
	ID    string            `yaml:"id,omitempty" json:"id,omitempty"`
	Uses  string            `yaml:"uses,omitempty" json:"uses,omitempty"`
	Run   string            `yaml:"run,omitempty" json:"run,omitempty"`
	With  map[string]string `yaml:"with,omitempty" json:"with,omitempty"`
	Env   map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	If    string            `yaml:"if,omitempty" json:"if,omitempty"`
	Shell string            `yaml:"shell,omitempty" json:"shell,omitempty"`
}

// Strategy defines the matrix strategy for a job.
type Strategy struct {
	Matrix      map[string][]string `yaml:"matrix" json:"matrix"`
	FailFast    *bool               `yaml:"fail-fast,omitempty" json:"fail-fast,omitempty"`
	MaxParallel int                 `yaml:"max-parallel,omitempty" json:"max-parallel,omitempty"`
}

// MatrixCombination represents one expanded combination from a matrix strategy.
type MatrixCombination map[string]string

// LintFinding represents a linting issue found in a workflow file.
type LintFinding struct {
	File     string `json:"file"`
	Line     int    `json:"line,omitempty"`
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// ValidationError represents a validation failure in a workflow.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error formats the error as "field: message" so that validation errors from
// multiple fields read well when concatenated.
func (v ValidationError) Error() string {
	return v.Field + ": " + v.Message
}
