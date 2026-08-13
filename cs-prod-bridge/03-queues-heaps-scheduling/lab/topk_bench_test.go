package lab

import (
	"fmt"
	"testing"
)

var benchmarkCandidates []Candidate

func BenchmarkTopK(b *testing.B) {
	cases := []struct {
		name       string
		candidates []Candidate
		k          int
	}{
		{name: "mixed/r-1000/k-10", candidates: mixedTopKFixture(1_000), k: 10},
		{name: "mixed/r-10000/k-10", candidates: mixedTopKFixture(10_000), k: 10},
		{name: "mixed/r-100000/k-10", candidates: mixedTopKFixture(100_000), k: 10},
		{name: "mixed/r-10000/k-1", candidates: mixedTopKFixture(10_000), k: 1},
		{name: "mixed/r-10000/k-1000", candidates: mixedTopKFixture(10_000), k: 1_000},
		{name: "mixed/r-10000/k-10000", candidates: mixedTopKFixture(10_000), k: 10_000},
		{name: "best-first/r-10000/k-100", candidates: orderedTopKFixture(10_000, true), k: 100},
		{name: "worst-first/r-10000/k-100", candidates: orderedTopKFixture(10_000, false), k: 100},
	}

	for _, benchmarkCase := range cases {
		b.Run(benchmarkCase.name, func(b *testing.B) {
			benchmarkTopKMethod(b, "sort-all", TopKBySort, benchmarkCase.candidates, benchmarkCase.k)
			benchmarkTopKMethod(b, "bounded-heap", TopKByHeap, benchmarkCase.candidates, benchmarkCase.k)
		})
	}
}

func benchmarkTopKMethod(
	b *testing.B,
	name string,
	selectTopK func([]Candidate, int) ([]Candidate, SelectionStats, error),
	candidates []Candidate,
	k int,
) {
	b.Run(name, func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var err error
			benchmarkCandidates, _, err = selectTopK(candidates, k)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func mixedTopKFixture(count int) []Candidate {
	candidates := make([]Candidate, count)
	for i := range candidates {
		// This permutation avoids handing either implementation presorted input.
		score := float64((i*7_919)%100_003) / 100_003
		candidates[i] = Candidate{
			Service: fmt.Sprintf("service-%08d", i),
			Score:   score,
		}
	}
	return candidates
}

func orderedTopKFixture(count int, bestFirst bool) []Candidate {
	candidates := make([]Candidate, count)
	for i := range candidates {
		score := i
		if bestFirst {
			score = count - i
		}
		candidates[i] = Candidate{
			Service: fmt.Sprintf("service-%08d", i),
			Score:   float64(score),
		}
	}
	return candidates
}
