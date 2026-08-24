// Package nestedloop models a batch flow-enrichment boundary.
package nestedloop

// Metadata associates an endpoint with its operational owner.
type Metadata struct {
	IP    string
	Owner string
}

// Event is an observed flow awaiting enrichment.
type Event struct {
	SourceIP string
	Bytes    int64
}

// EnrichedEvent records whether endpoint metadata was available.
type EnrichedEvent struct {
	Event    Event
	Metadata Metadata
	Found    bool
}

// Stats exposes deterministic work independent of wall-clock timing.
type Stats struct {
	CandidateChecks int
}

// Service enriches events from the metadata snapshot supplied with each batch.
type Service struct{}

// HandleBatch applies last-record-wins semantics for duplicate endpoint IPs.
func (Service) HandleBatch(events []Event, records []Metadata) ([]EnrichedEvent, Stats) {
	result := make([]EnrichedEvent, len(events))
	var stats Stats

	for i, event := range events {
		result[i].Event = event
		for j := len(records) - 1; j >= 0; j-- {
			stats.CandidateChecks++
			if records[j].IP != event.SourceIP {
				continue
			}
			result[i].Metadata = records[j]
			result[i].Found = true
			break
		}
	}
	return result, stats
}
