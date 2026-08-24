package lab

import (
	"fmt"
	"testing"
)

var (
	benchmarkFilter *Filter
	benchmarkRecord Record
	benchmarkFound  bool
	benchmarkStats  LookupStats
)

func BenchmarkBuildMembership(b *testing.B) {
	for _, itemCount := range []int{100, 1_000, 10_000} {
		b.Run(fmt.Sprintf("items-%d", itemCount), func(b *testing.B) {
			keys := keyBytes("build", itemCount)
			cfg := Config{
				PlannedItems:            uint64(itemCount),
				TargetFalsePositiveRate: 0.01,
			}
			filter, err := BuildMembership(keys, cfg)
			if err != nil {
				b.Fatalf("BuildMembership: %v", err)
			}
			filterBytes := filter.Stats().Bytes
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				benchmarkFilter, err = BuildMembership(keys, cfg)
				if err != nil {
					b.Fatalf("BuildMembership: %v", err)
				}
			}
			b.ReportMetric(float64(filterBytes), "filter-bytes")
		})
	}
}

func BenchmarkSegmentLookups(b *testing.B) {
	for _, shape := range []struct{ segments, keys int }{{4, 100}, {16, 1_000}, {32, 10_000}} {
		b.Run(fmt.Sprintf("segments-%d/keys-%d", shape.segments, shape.keys), func(b *testing.B) {
			b.StopTimer()
			filtered := makeSegments(b, shape.segments, shape.keys, 0.01)
			exact := make([]*Segment, len(filtered))
			for i := range exact {
				var err error
				exact[i], err = NewSegment(makeRecordsForSegment(i, shape.keys), nil)
				if err != nil {
					b.Fatalf("NewSegment(%d): %v", i, err)
				}
			}
			absentKeys := make([]string, 1_024)
			for i := range absentKeys {
				absentKeys[i] = fmt.Sprintf("absent-benchmark-key-%06d", i)
			}
			benchmarkAbsentLookups(b, "filtered-absent", filtered, absentKeys)
			benchmarkAbsentLookups(b, "exact-absent", exact, absentKeys)
		})
	}
}

func benchmarkAbsentLookups(b *testing.B, name string, segments []*Segment, absentKeys []string) {
	b.Run(name, func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			benchmarkRecord, benchmarkFound, benchmarkStats = LookupSegments(segments, absentKeys[i%len(absentKeys)])
		}
	})
}

func BenchmarkFilterCapacity(b *testing.B) {
	for _, multiplier := range []int{1, 2, 10} {
		b.Run(fmt.Sprintf("insertions-%dx-planned", multiplier), func(b *testing.B) {
			b.StopTimer()
			cfg := Config{PlannedItems: 1_000, TargetFalsePositiveRate: 0.01}
			segment, err := NewSegment(makeRecords("present", 1_000*multiplier), &cfg)
			if err != nil {
				b.Fatalf("NewSegment: %v", err)
			}
			absentKeys := make([]string, 10_000)
			for i := range absentKeys {
				absentKeys[i] = fmt.Sprintf("absent-%06d", i)
			}
			segments := []*Segment{segment}
			b.ReportMetric(segment.filter.Stats().BitDensity, "density")
			b.ReportAllocs()
			b.StartTimer()
			for i := 0; i < b.N; i++ {
				benchmarkRecord, benchmarkFound, benchmarkStats = LookupSegments(segments, absentKeys[i%len(absentKeys)])
			}
		})
	}
}
