package types

// Severity represents the severity level of an audit finding.
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarning
	SeverityCritical
)

func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// Finding represents a single audit finding for a Kubernetes resource.
type Finding struct {
	Resource    string            `json:"resource"`
	Namespace   string            `json:"namespace"`
	Name        string            `json:"name"`
	Severity    Severity          `json:"severity"`
	Message     string            `json:"message"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Report aggregates audit findings.
type Report struct {
	Findings []Finding `json:"findings"`
	Summary  Summary   `json:"summary"`
}

// Summary contains aggregate counts of findings by severity.
type Summary struct {
	Total    int `json:"total"`
	Critical int `json:"critical"`
	Warning  int `json:"warning"`
	Info     int `json:"info"`
}
