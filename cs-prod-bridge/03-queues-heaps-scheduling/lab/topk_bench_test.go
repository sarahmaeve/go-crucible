package lab

import (
	"fmt"
	"testing"
)

var benchmarkCandidates []Candidate

func BenchmarkTopK(b *testing.B) {
	for _, candidateCount := range []int{1_000, 10_000, 100_000} {
		candidates := topKFixture(candidateCount)
		for _, k := range []int{10, 100} {
			b.Run(fmt.Sprintf("candidates-%d/k-%d", candidateCount, k), func(b *testing.B) {
				b.Run("sort-all", func(b *testing.B) {
					b.ReportAllocs()
					for i := 0; i < b.N; i++ {
						var err error
						benchmarkCandidates, _, err = TopKBySort(candidates, k)
						if err != nil {
							b.Fatal(err)
						}
					}
				})
				b.Run("bounded-heap", func(b *testing.B) {
					b.ReportAllocs()
					for i := 0; i < b.N; i++ {
						var err error
						benchmarkCandidates, _, err = TopKByHeap(candidates, k)
						if err != nil {
							b.Fatal(err)
						}
					}
				})
			})
		}
	}
}

func topKFixture(count int) []Candidate {
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
