package lab

import (
	"fmt"
	"testing"
)

var (
	benchmarkIDs       []uint64
	benchmarkDocuments []Document
)

func BenchmarkSearch(b *testing.B) {
	workloads := []int{1_000, 10_000, 100_000}
	for _, documentCount := range workloads {
		documents := benchmarkSearchFixture(documentCount)
		index, err := BuildIndex(documents)
		if err != nil {
			b.Fatal(err)
		}
		terms := []string{"oncall", "rotation"}

		b.Run(fmt.Sprintf("documents-%d", documentCount), func(b *testing.B) {
			b.Run("scan", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					benchmarkIDs, _ = ScanAND(documents, terms)
				}
			})
			b.Run("index-end-to-end", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					built, buildErr := BuildIndex(documents)
					if buildErr != nil {
						b.Fatal(buildErr)
					}
					benchmarkIDs, _ = built.SearchAND(terms)
				}
			})
			b.Run("index-reused", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					benchmarkIDs, _ = index.SearchAND(terms)
				}
			})
		})
	}
}

func BenchmarkIndexReuse(b *testing.B) {
	documents := benchmarkSearchFixture(20_000)
	terms := []string{"oncall", "rotation"}
	for _, queryCount := range []int{1, 10, 100} {
		b.Run(fmt.Sprintf("queries-per-build-%d", queryCount), func(b *testing.B) {
			b.ReportAllocs()
			b.ReportMetric(float64(queryCount), "queries/build")
			for i := 0; i < b.N; i++ {
				index, err := BuildIndex(documents)
				if err != nil {
					b.Fatal(err)
				}
				for range queryCount {
					benchmarkIDs, _ = index.SearchAND(terms)
				}
			}
		})
	}
}

func BenchmarkSelectivity(b *testing.B) {
	documents := benchmarkSearchFixture(20_000)
	index, err := BuildIndex(documents)
	if err != nil {
		b.Fatal(err)
	}
	queries := []struct {
		name        string
		terms       []string
		wantLengths [2]int
		wantMatches int
	}{
		{
			name:        "rare-rare-disjoint",
			terms:       []string{"shard-997", "shard-991"},
			wantLengths: [2]int{20, 20},
			wantMatches: 0,
		},
		{
			name:        "rare-common-contained",
			terms:       []string{"shard-998", "oncall"},
			wantLengths: [2]int{20, 10_000},
			wantMatches: 20,
		},
		{
			name:        "common-common-partial-overlap",
			terms:       []string{"oncall", "rotation"},
			wantLengths: [2]int{10_000, 6_667},
			wantMatches: 3_334,
		},
	}
	for _, query := range queries {
		for i, term := range query.terms {
			if got := len(index.Postings(term)); got != query.wantLengths[i] {
				b.Fatalf("postings length for %q = %d, want %d", term, got, query.wantLengths[i])
			}
		}
		matches, _ := index.SearchAND(query.terms)
		if len(matches) != query.wantMatches {
			b.Fatalf("matches for %q = %d, want %d", query.name, len(matches), query.wantMatches)
		}
		b.Run(query.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchmarkIDs, _ = index.SearchAND(query.terms)
			}
		})
	}
}

func BenchmarkSortedUpdates(b *testing.B) {
	contiguous := makeRange(10_000, 20_000)
	orders := []struct {
		name    string
		current []uint64
		batch   []uint64
	}{
		{name: "front", current: contiguous, batch: makeRange(0, 1_000)},
		{name: "append", current: contiguous, batch: makeRange(20_000, 21_000)},
		{name: "uniform-interleaved", current: evenIDs(10_000), batch: distributedOddIDs(1_000)},
	}
	for _, order := range orders {
		b.Run(order.name, func(b *testing.B) {
			b.Run("individual", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					var updateErr error
					benchmarkIDs, _, updateErr = InsertIndividually(order.current, order.batch)
					if updateErr != nil {
						b.Fatal(updateErr)
					}
				}
			})
			b.Run("sort-and-merge", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					var updateErr error
					benchmarkIDs, _, updateErr = SortAndMergeBatch(order.current, order.batch)
					if updateErr != nil {
						b.Fatal(updateErr)
					}
				}
			})
		})
	}
}

func benchmarkSearchFixture(count int) []Document {
	documents := make([]Document, count)
	for i := range documents {
		tags := []string{
			fmt.Sprintf("region-%d", i%8),
			fmt.Sprintf("shard-%d", i%1_000),
		}
		if i%2 == 0 {
			tags = append(tags, "oncall")
		}
		if i%3 == 0 {
			tags = append(tags, "rotation")
		}
		documents[i] = Document{
			ID:        uint64(count - i),
			Title:     fmt.Sprintf("Runbook %d", i),
			Tags:      tags,
			UpdatedAt: int64(i / 4),
		}
	}
	return documents
}

func evenIDs(count int) []uint64 {
	result := make([]uint64, count)
	for i := range result {
		result[i] = uint64(i * 2)
	}
	return result
}

func distributedOddIDs(count int) []uint64 {
	result := make([]uint64, count)
	for i := range result {
		result[i] = uint64(i*20 + 1)
	}
	return result
}
