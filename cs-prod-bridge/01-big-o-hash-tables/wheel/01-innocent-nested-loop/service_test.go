package nestedloop

import "testing"

func TestHandleBatchPreservesEnrichmentSemantics(t *testing.T) {
	records := []Metadata{
		{IP: "10.0.0.1", Owner: "old-owner"},
		{IP: "10.0.0.2", Owner: "billing"},
		{IP: "10.0.0.1", Owner: "new-owner"},
	}
	events := []Event{
		{SourceIP: "10.0.0.1", Bytes: 100},
		{SourceIP: "192.0.2.1", Bytes: 200},
	}

	got, _ := (Service{}).HandleBatch(events, records)
	if !got[0].Found || got[0].Metadata.Owner != "new-owner" {
		t.Fatalf("duplicate policy: got %#v, want last record", got[0])
	}
	if got[1].Found {
		t.Fatalf("missing metadata: got %#v, want Found=false", got[1])
	}
	if got[1].Event != events[1] {
		t.Fatalf("missing event changed: got %#v, want %#v", got[1].Event, events[1])
	}
}
