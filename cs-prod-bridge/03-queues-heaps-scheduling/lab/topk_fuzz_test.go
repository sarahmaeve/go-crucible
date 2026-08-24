package lab

import (
	"math"
	"slices"
	"testing"
)

func FuzzTopKImplementationsAgree(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0})
	f.Add([]byte{3, 10, 1})
	f.Add([]byte{0, 10, 1, 10, 2, 9, 3, 8})
	f.Add([]byte{3, 0, 2, 1, 1, 2, 0, 3})
	f.Add([]byte{0, 1, 1, 2, 2, 3, 3, 4})

	f.Fuzz(func(t *testing.T, encoded []byte) {
		// Two bytes encode one candidate. A one-byte name deliberately creates
		// ties and duplicate records; the score byte covers NaN, infinities,
		// negative values, and many repeated finite scores.
		if len(encoded) > 128 {
			encoded = encoded[:128]
		}
		candidates := make([]Candidate, 0, len(encoded)/2)
		for i := 0; i+1 < len(encoded); i += 2 {
			candidates = append(candidates, Candidate{
				Service: string([]byte{encoded[i]}),
				Score:   fuzzScore(encoded[i+1]),
			})
		}

		// Derive k separately from the candidate pairs and include both invalid
		// and larger-than-input boundary values.
		k := 0
		if len(encoded)%2 == 1 {
			switch encoded[len(encoded)-1] % 4 {
			case 0:
				k = -1
			case 1:
				k = len(candidates) + 1
			default:
				if len(candidates) > 0 {
					k = int(encoded[len(encoded)-1]) % (len(candidates) + 1)
				}
			}
		} else if len(candidates) > 0 {
			k = int(encoded[0]) % (len(candidates) + 1)
		}

		original := slices.Clone(candidates)
		bySort, _, sortErr := TopKBySort(candidates, k)
		byHeap, heapStats, heapErr := TopKByHeap(candidates, k)

		if (sortErr == nil) != (heapErr == nil) {
			t.Fatalf("error mismatch for k=%d: sort=%v heap=%v", k, sortErr, heapErr)
		}
		if sortErr != nil && sortErr.Error() != heapErr.Error() {
			t.Fatalf("different errors for k=%d: sort=%q heap=%q", k, sortErr, heapErr)
		}
		if !sameCandidatesByBits(candidates, original) {
			t.Fatalf("input mutated: got %#v, want %#v", candidates, original)
		}
		if sortErr != nil {
			return
		}
		if !slices.Equal(byHeap, bySort) {
			t.Fatalf("TopKByHeap(%#v, %d) = %#v, TopKBySort = %#v", candidates, k, byHeap, bySort)
		}

		wantLen := min(k, len(candidates))
		if len(byHeap) != wantLen {
			t.Fatalf("result length = %d, want %d", len(byHeap), wantLen)
		}
		for i := 1; i < len(byHeap); i++ {
			if better(byHeap[i], byHeap[i-1]) {
				t.Fatalf("result is not in best-first order: %#v", byHeap)
			}
		}
		if heapStats.MaxRetained != wantLen {
			t.Fatalf("heap retained %d candidates, want %d", heapStats.MaxRetained, wantLen)
		}
	})
}

func fuzzScore(encoded byte) float64 {
	switch encoded {
	case 0:
		return math.NaN()
	case 1:
		return math.Inf(1)
	case 2:
		return math.Inf(-1)
	default:
		return float64(int8(encoded)) / 4
	}
}

func sameCandidatesByBits(left, right []Candidate) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].Service != right[i].Service || math.Float64bits(left[i].Score) != math.Float64bits(right[i].Score) {
			return false
		}
	}
	return true
}
