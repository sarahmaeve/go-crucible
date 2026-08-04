package lab

import (
	"reflect"
	"testing"
)

func TestEnrichmentImplementationsAgree(t *testing.T) {
	records := []Metadata{
		{IP: "10.0.0.1", Owner: "edge", Region: "us-west"},
		{IP: "10.0.0.2", Owner: "billing", Region: "us-east"},
	}
	events := []Event{
		{SourceIP: "10.0.0.2", Bytes: 2_048},
		{SourceIP: "10.0.0.99", Bytes: 512},
		{SourceIP: "10.0.0.1", Bytes: 1_024},
	}

	want := EnrichWithScan(events, records)
	got := NewMetadataIndex(records).Enrich(events)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("indexed enrichment differs from scan:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestEnrichmentUsesLastRecordForDuplicateIP(t *testing.T) {
	records := []Metadata{
		{IP: "10.0.0.1", Owner: "old-owner", Region: "us-west"},
		{IP: "10.0.0.1", Owner: "new-owner", Region: "us-west"},
	}
	events := []Event{{SourceIP: "10.0.0.1"}}

	for name, enrich := range map[string]func() []EnrichedEvent{
		"scan":  func() []EnrichedEvent { return EnrichWithScan(events, records) },
		"index": func() []EnrichedEvent { return NewMetadataIndex(records).Enrich(events) },
	} {
		t.Run(name, func(t *testing.T) {
			got := enrich()
			if !got[0].Found || got[0].Metadata.Owner != "new-owner" {
				t.Fatalf("got %#v, want last duplicate record", got[0])
			}
		})
	}
}

func TestEnrichmentReportsMissingMetadata(t *testing.T) {
	events := []Event{{SourceIP: "192.0.2.1", Bytes: 100}}

	got := NewMetadataIndex(nil).Enrich(events)
	if got[0].Found {
		t.Fatalf("got %#v, want missing metadata", got[0])
	}
	if got[0].Event != events[0] {
		t.Fatalf("event changed: got %#v, want %#v", got[0].Event, events[0])
	}
}
