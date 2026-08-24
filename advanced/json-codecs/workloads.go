package jsoncodecs

import (
	"fmt"
	"time"
)

// HealthStatus models a small response where codec choice is unlikely to
// dominate total request cost.
type HealthStatus struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

// FlatRecord separates fixed-schema codec work from map-heavy workloads.
type FlatRecord struct {
	ID          int64   `json:"id"`
	Service     string  `json:"service"`
	Environment string  `json:"environment"`
	Region      string  `json:"region"`
	Method      string  `json:"method"`
	Path        string  `json:"path"`
	StatusCode  int     `json:"status_code"`
	DurationMS  float64 `json:"duration_ms"`
	Cached      bool    `json:"cached"`
}

// FlatBatch contains no maps, interfaces, or custom marshal methods.
type FlatBatch struct {
	Records []FlatRecord `json:"records"`
}

// Metric models one item in a medium-sized ingestion request.
type Metric struct {
	Name      string            `json:"name"`
	Labels    map[string]string `json:"labels"`
	Value     float64           `json:"value"`
	Timestamp time.Time         `json:"timestamp"`
}

// MetricBatch models a typed request body.
type MetricBatch struct {
	Metrics []Metric `json:"metrics"`
}

// Finding models one item in a potentially large API response.
type Finding struct {
	Resource    string            `json:"resource"`
	Namespace   string            `json:"namespace"`
	Name        string            `json:"name"`
	Severity    string            `json:"severity"`
	Message     string            `json:"message"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// FindingList models a whole buffered list response.
type FindingList struct {
	Findings []Finding `json:"findings"`
	Total    int       `json:"total"`
}

// Workload pairs one named value with the encoded bytes used for decoding.
type Workload struct {
	Name  string
	Value any
	JSON  []byte
	New   func() any
}

// RepresentativeWorkloads returns deterministic small, medium, and large
// payloads. The standard library creates the baseline bytes so all decoders
// receive identical input.
func RepresentativeWorkloads() ([]Workload, error) {
	values := []struct {
		name  string
		value any
		new   func() any
	}{
		{name: "small-health", value: HealthStatus{Status: "ready"}, new: func() any { return new(HealthStatus) }},
		{name: "medium-flat", value: makeFlatBatch(256), new: func() any { return new(FlatBatch) }},
		{name: "medium-metrics", value: makeMetricBatch(128), new: func() any { return new(MetricBatch) }},
		{name: "large-findings", value: makeFindingList(2_048), new: func() any { return new(FindingList) }},
	}

	workloads := make([]Workload, 0, len(values))
	for _, value := range values {
		data, err := StandardLibrary.Marshal(value.value)
		if err != nil {
			return nil, fmt.Errorf("marshal %s fixture: %w", value.name, err)
		}
		workloads = append(workloads, Workload{Name: value.name, Value: value.value, JSON: data, New: value.new})
	}
	return workloads, nil
}

func findWorkload(workloads []Workload, name string) (Workload, bool) {
	for _, workload := range workloads {
		if workload.Name == name {
			return workload, true
		}
	}
	return Workload{}, false
}

func makeFlatBatch(count int) FlatBatch {
	records := make([]FlatRecord, count)
	for i := range records {
		records[i] = FlatRecord{
			ID:          int64(10_000_000 + i),
			Service:     fmt.Sprintf("api-%02d", i%32),
			Environment: "production",
			Region:      fmt.Sprintf("region-%d", i%4),
			Method:      []string{"GET", "POST", "PUT", "DELETE"}[i%4],
			Path:        fmt.Sprintf("/v1/resources/%06d", i),
			StatusCode:  []int{200, 201, 404, 503}[i%4],
			DurationMS:  float64(i%10_000) / 100,
			Cached:      i%3 == 0,
		}
	}
	return FlatBatch{Records: records}
}

func makeMetricBatch(count int) MetricBatch {
	metrics := make([]Metric, count)
	base := time.Unix(1_700_000_000, 0).UTC()
	for i := range metrics {
		metrics[i] = Metric{
			Name:      fmt.Sprintf("service.request.duration.%02d", i%16),
			Labels:    map[string]string{"environment": "production", "region": fmt.Sprintf("region-%d", i%4), "service": fmt.Sprintf("api-%02d", i%32)},
			Value:     float64(i%1_000) / 10,
			Timestamp: base.Add(time.Duration(i) * time.Second),
		}
	}
	return MetricBatch{Metrics: metrics}
}

func makeFindingList(count int) FindingList {
	findings := make([]Finding, count)
	for i := range findings {
		findings[i] = Finding{
			Resource:  "deployments",
			Namespace: fmt.Sprintf("tenant-%03d", i%200),
			Name:      fmt.Sprintf("workload-%06d", i),
			Severity:  []string{"info", "warning", "critical"}[i%3],
			Message:   "container has no explicit CPU or memory limit; review workload policy before deployment",
			Labels:    map[string]string{"app": fmt.Sprintf("service-%03d", i%128), "team": fmt.Sprintf("platform-%02d", i%12)},
			Annotations: map[string]string{
				"documentation": "https://example.invalid/policies/resources?kind=deployment&source=crucible",
			},
		}
	}
	return FindingList{Findings: findings, Total: len(findings)}
}
