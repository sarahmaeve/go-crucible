//go:build csbridgewheel1

package nestedloop

import (
	"fmt"
	"testing"
)

var wheel1Result []EnrichedEvent

func TestProductionScale(t *testing.T) {
	events, records := wheel1Fixture(4_000, 18_400)

	_, stats := (Service{}).HandleBatch(events, records)
	maxExpectedChecks := len(events) * 2
	if stats.CandidateChecks > maxExpectedChecks {
		t.Fatalf("performed %d candidate checks for %d events and %d metadata records; "+
			"want at most %d checks after constructing one batch index",
			stats.CandidateChecks, len(events), len(records), maxExpectedChecks)
	}
}

func BenchmarkHandleBatch(b *testing.B) {
	events, records := wheel1Fixture(4_000, 18_400)
	service := Service{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wheel1Result, _ = service.HandleBatch(events, records)
	}
}

func wheel1Fixture(eventCount, metadataCount int) ([]Event, []Metadata) {
	records := make([]Metadata, metadataCount)
	for i := range records {
		records[i] = Metadata{
			IP:    fmt.Sprintf("10.%d.%d.%d", i>>16, i>>8&0xff, i&0xff),
			Owner: fmt.Sprintf("service-%d", i%128),
		}
	}

	events := make([]Event, eventCount)
	for i := range events {
		record := records[(i*97)%len(records)]
		events[i] = Event{SourceIP: record.IP, Bytes: int64(1_024 + i%4_096)}
	}
	return events, records
}
