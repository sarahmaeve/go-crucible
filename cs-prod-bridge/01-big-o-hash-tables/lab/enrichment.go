// Package lab compares scan-based and indexed event enrichment.
package lab

// Metadata associates an IP address with operational ownership information.
type Metadata struct {
	IP     string
	Owner  string
	Region string
}

// Event is a network observation awaiting metadata enrichment.
type Event struct {
	SourceIP string
	Bytes    int64
}

// EnrichedEvent records both the original event and whether metadata existed.
type EnrichedEvent struct {
	Event    Event
	Metadata Metadata
	Found    bool
}

// MetadataIndex supports exact-IP metadata lookup.
type MetadataIndex map[string]Metadata

// NewMetadataIndex builds an index using a last-record-wins duplicate policy.
func NewMetadataIndex(records []Metadata) MetadataIndex {
	index := make(MetadataIndex, len(records))
	for _, record := range records {
		index[record.IP] = record
	}
	return index
}

// EnrichWithScan enriches each event by scanning records from newest to oldest.
// Scanning backward implements the same last-record-wins duplicate policy as
// NewMetadataIndex.
func EnrichWithScan(events []Event, records []Metadata) []EnrichedEvent {
	result := make([]EnrichedEvent, len(events))
	for i, event := range events {
		result[i].Event = event
		for j := len(records) - 1; j >= 0; j-- {
			if records[j].IP != event.SourceIP {
				continue
			}
			result[i].Metadata = records[j]
			result[i].Found = true
			break
		}
	}
	return result
}

// Enrich enriches each event with an exact-key index lookup.
func (index MetadataIndex) Enrich(events []Event) []EnrichedEvent {
	result := make([]EnrichedEvent, len(events))
	for i, event := range events {
		metadata, ok := index[event.SourceIP]
		result[i] = EnrichedEvent{
			Event:    event,
			Metadata: metadata,
			Found:    ok,
		}
	}
	return result
}
