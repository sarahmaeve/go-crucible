package cachewithouthits

import (
	"errors"
	"testing"
	"time"
)

type fakeLoader struct {
	metadata map[string]Metadata
	calls    int
	err      error
}

func (l *fakeLoader) LoadEndpoint(ip string) (Metadata, error) {
	l.calls++
	if l.err != nil {
		return Metadata{}, l.err
	}
	return l.metadata[ip], nil
}

func TestResolverEnrichesAndReusesIdenticalObservation(t *testing.T) {
	loader := &fakeLoader{metadata: map[string]Metadata{
		"10.0.0.1": {Owner: "payments", Region: "us-west"},
	}}
	resolver := NewResolver(loader)
	observedAt := time.Unix(1_700_000_000, 0)
	observations := []Observation{
		{IP: "10.0.0.1", ObservedAt: observedAt},
		{IP: "10.0.0.1", ObservedAt: observedAt},
	}

	got, err := resolver.Resolve(observations)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	if loader.calls != 1 {
		t.Fatalf("loader calls = %d, want 1 for identical observation", loader.calls)
	}
	if got[0].Metadata.Owner != "payments" || got[1].Metadata != got[0].Metadata {
		t.Fatalf("unexpected metadata: %#v", got)
	}
}

func TestResolverReturnsLoaderError(t *testing.T) {
	wantErr := errors.New("metadata unavailable")
	resolver := NewResolver(&fakeLoader{err: wantErr})

	_, err := resolver.Resolve([]Observation{{IP: "10.0.0.1", ObservedAt: time.Now()}})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Resolve() error = %v, want %v", err, wantErr)
	}
	if resolver.EntryCount() != 0 {
		t.Fatalf("cache entries = %d after failed load, want 0", resolver.EntryCount())
	}
}
