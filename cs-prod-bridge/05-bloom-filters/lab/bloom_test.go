package lab

import (
	"fmt"
	"hash/maphash"
	"math"
	"sync"
	"testing"
)

func TestBuildMembershipRejectsInvalidConfig(t *testing.T) {
	for _, cfg := range []Config{
		{},
		{PlannedItems: 10},
		{PlannedItems: 10, TargetFalsePositiveRate: 1},
		{PlannedItems: 10, TargetFalsePositiveRate: math.NaN()},
	} {
		if _, err := BuildMembership(nil, cfg); err == nil {
			t.Fatalf("BuildMembership(nil, %#v) succeeded", cfg)
		}
	}
}

func TestBuildMembershipAcceptsBoundaryInputsAndRejectsDuplicates(t *testing.T) {
	tests := []struct {
		name string
		keys [][]byte
		cfg  Config
	}{
		{
			name: "empty set",
			cfg:  Config{PlannedItems: 1, TargetFalsePositiveRate: 0.01},
		},
		{
			name: "one item",
			keys: keyBytes("one", 1),
			cfg:  Config{PlannedItems: 1, TargetFalsePositiveRate: 0.01},
		},
		{
			name: "multiple words",
			keys: keyBytes("many", 100),
			cfg:  Config{PlannedItems: 100, TargetFalsePositiveRate: 0.01},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := buildDeterministicMembership(tt.keys, tt.cfg)
			if err != nil {
				t.Fatalf("buildDeterministicMembership(%q): %v", tt.name, err)
			}
			for _, key := range tt.keys {
				if !filter.MayContain(key) {
					t.Errorf("MayContain(%q) = false, want true", key)
				}
			}
			if tt.name == "multiple words" {
				nonzeroWords := 0
				for _, word := range filter.words {
					if word != 0 {
						nonzeroWords++
					}
				}
				if nonzeroWords < 2 {
					t.Errorf("nonzero filter words = %d, want at least 2", nonzeroWords)
				}
			}
		})
	}

	duplicate := [][]byte{[]byte("same"), []byte("same")}
	cfg := Config{PlannedItems: 2, TargetFalsePositiveRate: 0.01}
	if _, err := BuildMembership(duplicate, cfg); err == nil {
		t.Fatal("BuildMembership accepted duplicate keys")
	}
	if _, err := BuildMembership([][]byte{nil, {}}, cfg); err == nil {
		t.Fatal("BuildMembership accepted nil and empty as distinct keys")
	}
}

func TestBuildMembershipEnforcesMaximumBytes(t *testing.T) {
	cfg := Config{
		PlannedItems:            1_000,
		TargetFalsePositiveRate: 0.01,
		MaxBytes:                8,
	}
	if _, err := BuildMembership(keyBytes("key", 1_000), cfg); err == nil {
		t.Fatalf("BuildMembership(%#v) succeeded, want allocation-limit error", cfg)
	}
}

func TestInsertedKeysNeverReturnNegative(t *testing.T) {
	keys := keyBytes("live", 1_000)
	filter, err := BuildMembership(keys, Config{PlannedItems: 1_000, TargetFalsePositiveRate: 0.01})
	if err != nil {
		t.Fatalf("BuildMembership: %v", err)
	}
	for _, key := range keys {
		if !filter.MayContain(key) {
			t.Fatalf("inserted key %q returned definitely absent", key)
		}
	}
	stats := filter.Stats()
	if stats.InsertedItems != 1_000 || stats.PlannedItems != 1_000 || stats.ProbeCount != 7 {
		t.Fatalf("filter stats = %#v", stats)
	}
	if stats.Bytes != int(stats.BitCount/8) || stats.BitDensity <= 0 || stats.BitDensity >= 1 {
		t.Fatalf("invalid representation stats: %#v", stats)
	}
	if stats.TargetFalsePositiveRate != 0.01 || stats.ModeledFalsePositiveRate <= 0 || stats.ModeledFalsePositiveRate > 0.01 {
		t.Fatalf("false-positive model stats = %#v", stats)
	}
	if stats.MaximumBytes != DefaultMaxFilterBytes {
		t.Fatalf("maximum bytes = %d, want %d", stats.MaximumBytes, DefaultMaxFilterBytes)
	}
}

func TestImmutableFilterSupportsConcurrentReads(t *testing.T) {
	keys := keyBytes("concurrent", 1_000)
	filter, err := BuildMembership(keys, Config{PlannedItems: 1_000, TargetFalsePositiveRate: 0.01})
	if err != nil {
		t.Fatalf("BuildMembership: %v", err)
	}

	const readers = 8
	errs := make(chan error, readers)
	var wg sync.WaitGroup
	wg.Add(readers)
	for reader := 0; reader < readers; reader++ {
		go func(reader int) {
			defer wg.Done()
			for i, key := range keys {
				if !filter.MayContain(key) {
					errs <- fmt.Errorf("reader %d: MayContain(%q) = false, want true", reader, key)
					return
				}
				filter.MayContain([]byte(fmt.Sprintf("absent-%d-%d", reader, i)))
			}
		}(reader)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

func TestFilteredAndExactSegmentsReturnTheSameAnswers(t *testing.T) {
	records := []Record{
		{Key: "alpha", Value: "A"},
		{Key: "charlie", Value: "C"},
		{Key: "echo", Value: "E"},
	}
	cfg := Config{PlannedItems: 3, TargetFalsePositiveRate: 0.05}
	filtered, err := NewSegment(records, &cfg)
	if err != nil {
		t.Fatalf("NewSegment(filtered): %v", err)
	}
	exact, err := NewSegment(records, nil)
	if err != nil {
		t.Fatalf("NewSegment(exact): %v", err)
	}
	for _, key := range []string{"alpha", "bravo", "charlie", "delta", "echo", "foxtrot"} {
		gotRecord, gotFound, _ := LookupSegments([]*Segment{filtered}, key)
		wantRecord, wantFound, _ := LookupSegments([]*Segment{exact}, key)
		if gotFound != wantFound || gotRecord != wantRecord {
			t.Fatalf("lookup %q with filter = (%#v, %t), exact = (%#v, %t)", key, gotRecord, gotFound, wantRecord, wantFound)
		}
	}
}

func TestAbsentWorkloadAccountingSeparatesFalsePositives(t *testing.T) {
	segments := makeSegments(t, 4, 250, 0.01)
	var total LookupStats
	for i := 0; i < 500; i++ {
		_, found, stats := LookupSegments(segments, fmt.Sprintf("absent-%06d", i))
		if found {
			t.Fatal("absent key was returned as an exact hit")
		}
		total.Add(stats)
	}
	want := uint64(4 * 500)
	if total.CandidateSegments != want || total.FilterQueries != want {
		t.Fatalf("candidate/filter queries = %d/%d, want %d", total.CandidateSegments, total.FilterQueries, want)
	}
	if total.ExactChecks+total.ExactChecksAvoided != want {
		t.Fatalf("exact checks %d + avoided %d != candidates %d", total.ExactChecks, total.ExactChecksAvoided, want)
	}
	if total.FalsePositives != total.ExactChecks || total.PossibleMatches != total.ExactChecks || total.TrueHits != 0 || total.ExactChecksAvoided == 0 {
		t.Fatalf("absent workload stats = %#v", total)
	}
}

func TestOverCapacityRaisesDensityWithoutBreakingCorrectness(t *testing.T) {
	cfg := Config{PlannedItems: 100, TargetFalsePositiveRate: 0.01}
	allKeys := keyBytes("key", 1_000)
	seed1, seed2 := maphash.MakeSeed(), maphash.MakeSeed()
	planned, err := buildMembership(allKeys[:100], cfg, seed1, seed2)
	if err != nil {
		t.Fatalf("buildMembership(planned): %v", err)
	}
	overloaded, err := buildMembership(allKeys, cfg, seed1, seed2)
	if err != nil {
		t.Fatalf("buildMembership(overloaded): %v", err)
	}
	if planned.Stats().BitCount != overloaded.Stats().BitCount {
		t.Fatal("exceeding capacity changed the fixed bit budget")
	}
	if overloaded.Stats().BitDensity <= planned.Stats().BitDensity {
		t.Fatalf("density did not rise: planned %.3f, overloaded %.3f", planned.Stats().BitDensity, overloaded.Stats().BitDensity)
	}
	for _, key := range allKeys {
		if !overloaded.MayContain(key) {
			t.Fatalf("overloaded filter lost inserted key %q", key)
		}
	}
	plannedPositives, overloadedPositives := 0, 0
	for i := 0; i < 2_000; i++ {
		key := []byte(fmt.Sprintf("absent-%06d", i))
		if planned.MayContain(key) {
			plannedPositives++
		}
		if overloaded.MayContain(key) {
			overloadedPositives++
		}
	}
	if overloadedPositives <= plannedPositives {
		t.Fatalf("possible matches did not rise: planned %d, overloaded %d", plannedPositives, overloadedPositives)
	}
}

func TestRepeatedFalsePositiveRepeatsExactWork(t *testing.T) {
	cfg := Config{PlannedItems: 8, TargetFalsePositiveRate: 0.20}
	segment, err := NewSegment(makeRecords("present", 80), &cfg)
	if err != nil {
		t.Fatalf("NewSegment: %v", err)
	}
	var hot string
	for i := 0; i < 100_000; i++ {
		candidate := fmt.Sprintf("missing-%06d", i)
		if segment.filter.MayContain([]byte(candidate)) {
			hot = candidate
			break
		}
	}
	if hot == "" {
		t.Fatal("could not find a false-positive key")
	}
	var total LookupStats
	for i := 0; i < 100; i++ {
		_, found, stats := LookupSegments([]*Segment{segment}, hot)
		if found {
			t.Fatal("false positive became an exact hit")
		}
		total.Add(stats)
	}
	if total.FalsePositives != 100 || total.ExactChecks != 100 {
		t.Fatalf("repeated false-positive stats = %#v", total)
	}
}

func TestLookupStopsAtFirstExactHit(t *testing.T) {
	segments := makeSegments(t, 5, 20, 0.01)
	record, found, stats := LookupSegments(segments, "segment-02-key-000007")
	if !found || record.Value != "value-02-000007" {
		t.Fatalf("lookup = %#v, %t", record, found)
	}
	if stats.CandidateSegments != 3 || stats.TrueHits != 1 || stats.ExactChecks == 0 {
		t.Fatalf("hit stats = %#v", stats)
	}
}

func TestNewSegmentCopiesInputAndRejectsDuplicates(t *testing.T) {
	records := []Record{{Key: "b", Value: "B"}, {Key: "a", Value: "A"}}
	segment, err := NewSegment(records, nil)
	if err != nil {
		t.Fatalf("NewSegment: %v", err)
	}
	records[0] = Record{Key: "changed"}
	if record, found, _ := LookupSegments([]*Segment{segment}, "b"); !found || record.Value != "B" {
		t.Fatal("segment retained caller-owned storage")
	}
	if _, err := NewSegment([]Record{{Key: "a"}, {Key: "a"}}, nil); err == nil {
		t.Fatal("duplicate keys were accepted")
	}
}

func makeSegments(t testing.TB, segmentCount, keysPerSegment int, target float64) []*Segment {
	t.Helper()
	segments := make([]*Segment, segmentCount)
	for segmentIndex := range segments {
		cfg := Config{PlannedItems: uint64(keysPerSegment), TargetFalsePositiveRate: target}
		var err error
		segments[segmentIndex], err = NewSegment(makeRecordsForSegment(segmentIndex, keysPerSegment), &cfg)
		if err != nil {
			t.Fatalf("NewSegment(%d): %v", segmentIndex, err)
		}
	}
	return segments
}

func makeRecords(prefix string, count int) []Record {
	records := make([]Record, count)
	for i := range records {
		records[i] = Record{
			Key:   fmt.Sprintf("%s-%06d", prefix, i),
			Value: fmt.Sprintf("value-%06d", i),
		}
	}
	return records
}

func makeRecordsForSegment(segmentIndex, count int) []Record {
	records := make([]Record, count)
	for i := range records {
		records[i] = Record{
			Key:   fmt.Sprintf("segment-%02d-key-%06d", segmentIndex, i),
			Value: fmt.Sprintf("value-%02d-%06d", segmentIndex, i),
		}
	}
	return records
}

func keyBytes(prefix string, count int) [][]byte {
	keys := make([][]byte, count)
	for i := range keys {
		keys[i] = []byte(fmt.Sprintf("%s-%06d", prefix, i))
	}
	return keys
}
