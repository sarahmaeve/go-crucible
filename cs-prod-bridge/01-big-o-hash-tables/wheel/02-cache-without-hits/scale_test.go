//go:build csbridgewheel2

package cachewithouthits

import (
	"fmt"
	"testing"
	"time"
)

var wheel2Result []EnrichedObservation

func TestRepeatedObservationsReuseEndpointMetadata(t *testing.T) {
	const (
		endpoints = 100
		rounds    = 200
	)
	loader, observations := wheel2Fixture(endpoints, rounds)
	resolver := NewResolver(loader)

	_, err := resolver.Resolve(observations)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	if loader.calls > endpoints {
		t.Errorf("loader calls = %d for %d stable endpoints, want at most %d",
			loader.calls, endpoints, endpoints)
	}
	if resolver.EntryCount() > endpoints {
		t.Errorf("cache entries = %d for %d stable endpoints, want at most %d",
			resolver.EntryCount(), endpoints, endpoints)
	}
}

func BenchmarkResolveRepeatedEndpoints(b *testing.B) {
	loader, observations := wheel2Fixture(600, 100)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolver := NewResolver(loader)
		var err error
		wheel2Result, err = resolver.Resolve(observations)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func wheel2Fixture(endpoints, rounds int) (*fakeLoader, []Observation) {
	metadata := make(map[string]Metadata, endpoints)
	for endpoint := 0; endpoint < endpoints; endpoint++ {
		ip := fmt.Sprintf("10.%d.%d.%d", endpoint>>16, endpoint>>8&0xff, endpoint&0xff)
		metadata[ip] = Metadata{
			Owner:  fmt.Sprintf("service-%d", endpoint%64),
			Region: fmt.Sprintf("region-%d", endpoint%4),
		}
	}

	base := time.Unix(1_700_000_000, 0)
	observations := make([]Observation, 0, endpoints*rounds)
	for round := 0; round < rounds; round++ {
		for endpoint := 0; endpoint < endpoints; endpoint++ {
			ip := fmt.Sprintf("10.%d.%d.%d", endpoint>>16, endpoint>>8&0xff, endpoint&0xff)
			observations = append(observations, Observation{
				IP:         ip,
				ObservedAt: base.Add(time.Duration(round*endpoints+endpoint) * time.Millisecond),
			})
		}
	}
	return &fakeLoader{metadata: metadata}, observations
}
