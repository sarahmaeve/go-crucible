package types

// Severity represents the severity level of an audit finding.
type Severity int

// Severity levels, ordered from least to most urgent.
const (
	// SeverityInfo is an informational finding that does not indicate a
	// problem, but is worth surfacing to an operator.
	SeverityInfo Severity = iota

	// SeverityWarning indicates a misconfiguration or risk that should be
	// addressed but is not an active incident.
	SeverityWarning

	// SeverityCritical indicates a live failure, security issue, or other
	// condition that should page a human.
	SeverityCritical
)

// String returns the lowercase name of the severity ("info", "warning",
// "critical"), or "unknown" for out-of-range values.
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
