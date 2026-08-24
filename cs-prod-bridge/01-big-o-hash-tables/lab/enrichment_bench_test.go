package lab

import (
	"fmt"
	"testing"
)

var benchmarkResult []EnrichedEvent

func BenchmarkEnrichment(b *testing.B) {
	workloads := []struct {
		name     string
		events   int
		metadata int
	}{
		{name: "small", events: 32, metadata: 16},
		{name: "medium", events: 1_024, metadata: 256},
		{name: "large", events: 4_096, metadata: 4_096},
	}

	for _, workload := range workloads {
		events, records := benchmarkFixture(workload.events, workload.metadata)
		b.Run(workload.name, func(b *testing.B) {
			b.Run("scan", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					benchmarkResult = EnrichWithScan(events, records)
				}
			})

			b.Run("index-end-to-end", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					index := NewMetadataIndex(records)
					benchmarkResult = index.Enrich(events)
				}
			})

			index := NewMetadataIndex(records)
			b.Run("index-reused", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					benchmarkResult = index.Enrich(events)
				}
			})
		})
	}
}

func benchmarkFixture(eventCount, metadataCount int) ([]Event, []Metadata) {
	records := make([]Metadata, metadataCount)
	for i := range records {
		records[i] = Metadata{
			IP:     fmt.Sprintf("10.%d.%d.%d", i>>16, i>>8&0xff, i&0xff),
			Owner:  fmt.Sprintf("service-%d", i%64),
			Region: fmt.Sprintf("region-%d", i%4),
		}
	}

	events := make([]Event, eventCount)
	for i := range events {
		record := records[(i*97)%len(records)]
		events[i] = Event{SourceIP: record.IP, Bytes: int64(512 + i%8_192)}
	}
	return events, records
}
