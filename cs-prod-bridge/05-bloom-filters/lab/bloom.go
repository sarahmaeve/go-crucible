// Package lab implements an in-memory Bloom filter and immutable-segment
// lookup path for exploring the cost of negative lookups.
package lab

import (
	"fmt"
	"hash/maphash"
	"math"
	"math/bits"
	"slices"
	"sort"
)

// DefaultMaxFilterBytes bounds allocations when Config.MaxBytes is zero.
const DefaultMaxFilterBytes uint64 = 64 << 20

// Membership is an approximate, read-only view of one set. False means the
// key is absent. True means that the caller must consult the exact source.
type Membership interface {
	MayContain(key []byte) bool
}

// Config sizes a filter for PlannedItems and TargetFalsePositiveRate. MaxBytes
// bounds the allocation; zero selects DefaultMaxFilterBytes. The target is a
// sizing input, not a promise about every finite query workload.
type Config struct {
	PlannedItems            uint64
	TargetFalsePositiveRate float64
	MaxBytes                uint64
}

// FilterStats describes the immutable filter that was built.
type FilterStats struct {
	PlannedItems             uint64
	InsertedItems            uint64
	BitCount                 uint64
	ProbeCount               uint64
	Bytes                    int
	MaximumBytes             uint64
	SetBits                  uint64
	BitDensity               float64
	TargetFalsePositiveRate  float64
	ModeledFalsePositiveRate float64
}

// Filter is an immutable, process-local Bloom filter. Its maphash seeds cannot
// be serialized or recreated in another process, so Filter is not a persistent
// storage format.
type Filter struct {
	words         []uint64
	probes        uint64
	seed1         maphash.Seed
	seed2         maphash.Seed
	deterministic bool
	stats         FilterStats
}

// BuildMembership builds one process-local filter from distinct supplied keys.
// It returns an error when keys contains a duplicate or cfg is invalid.
func BuildMembership(keys [][]byte, cfg Config) (*Filter, error) {
	return buildMembership(keys, cfg, maphash.MakeSeed(), maphash.MakeSeed())
}

func buildMembership(keys [][]byte, cfg Config, seed1, seed2 maphash.Seed) (*Filter, error) {
	return buildMembershipMode(keys, cfg, seed1, seed2, false)
}

func buildDeterministicMembership(keys [][]byte, cfg Config) (*Filter, error) {
	return buildMembershipMode(keys, cfg, maphash.Seed{}, maphash.Seed{}, true)
}

func buildMembershipMode(keys [][]byte, cfg Config, seed1, seed2 maphash.Seed, deterministic bool) (*Filter, error) {
	bitCount, probeCount, err := dimensions(cfg)
	if err != nil {
		return nil, err
	}
	wordCount := (bitCount + 63) / 64
	maxInt := uint64(int(^uint(0) >> 1))
	if wordCount > maxInt {
		return nil, fmt.Errorf("filter requires too many words: %d", wordCount)
	}
	requiredBytes := wordCount * 8
	if requiredBytes > maxInt {
		return nil, fmt.Errorf("filter requires too many bytes: %d", requiredBytes)
	}
	maximumBytes := cfg.MaxBytes
	if maximumBytes == 0 {
		maximumBytes = DefaultMaxFilterBytes
	}
	if requiredBytes > maximumBytes {
		return nil, fmt.Errorf("filter requires %d bytes; maximum is %d", requiredBytes, maximumBytes)
	}
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		canonicalKey := string(key)
		if _, ok := seen[canonicalKey]; ok {
			return nil, fmt.Errorf("duplicate membership key %q", key)
		}
		seen[canonicalKey] = struct{}{}
	}

	f := &Filter{
		words:         make([]uint64, int(wordCount)),
		probes:        probeCount,
		seed1:         seed1,
		seed2:         seed2,
		deterministic: deterministic,
		stats: FilterStats{
			PlannedItems:            cfg.PlannedItems,
			InsertedItems:           uint64(len(keys)),
			BitCount:                wordCount * 64,
			ProbeCount:              probeCount,
			Bytes:                   int(requiredBytes),
			MaximumBytes:            maximumBytes,
			TargetFalsePositiveRate: cfg.TargetFalsePositiveRate,
		},
	}
	for _, key := range keys {
		f.add(key)
	}
	for _, word := range f.words {
		f.stats.SetBits += uint64(bits.OnesCount64(word))
	}
	f.stats.BitDensity = float64(f.stats.SetBits) / float64(f.stats.BitCount)
	f.stats.ModeledFalsePositiveRate = modeledFalsePositiveRate(
		f.stats.InsertedItems,
		f.stats.BitCount,
		f.stats.ProbeCount,
	)
	return f, nil
}

// MayContain reports whether every derived position is set.
func (f *Filter) MayContain(key []byte) bool {
	possible, _ := f.query(key)
	return possible
}

// Stats returns sizing and occupancy information for the filter.
func (f *Filter) Stats() FilterStats { return f.stats }

func (f *Filter) add(key []byte) {
	h1, h2 := f.hashes(key)
	for i := uint64(0); i < f.probes; i++ {
		position := (h1 + i*h2) % f.stats.BitCount
		f.words[position/64] |= uint64(1) << (position % 64)
	}
}

func (f *Filter) query(key []byte) (bool, uint64) {
	h1, h2 := f.hashes(key)
	for i := uint64(0); i < f.probes; i++ {
		position := (h1 + i*h2) % f.stats.BitCount
		if f.words[position/64]&(uint64(1)<<(position%64)) == 0 {
			return false, i + 1
		}
	}
	return true, f.probes
}

func (f *Filter) hashes(key []byte) (uint64, uint64) {
	var h1, h2 uint64
	if f.deterministic {
		h1, h2 = stableHashPair(key)
	} else {
		h1, h2 = maphash.Bytes(f.seed1, key), maphash.Bytes(f.seed2, key)
	}
	// The bit count is a multiple of 64. An odd step traverses that space
	// instead of falling into a shorter even-numbered cycle.
	return h1, h2 | 1
}

func dimensions(cfg Config) (uint64, uint64, error) {
	if cfg.PlannedItems == 0 {
		return 0, 0, fmt.Errorf("planned items must be positive")
	}
	p := cfg.TargetFalsePositiveRate
	if math.IsNaN(p) || math.IsInf(p, 0) || p <= 0 || p >= 1 {
		return 0, 0, fmt.Errorf("target false-positive rate must be between 0 and 1")
	}
	m := -float64(cfg.PlannedItems) * math.Log(p) / (math.Ln2 * math.Ln2)
	if math.IsInf(m, 0) || m > float64(^uint64(0)-63) {
		return 0, 0, fmt.Errorf("requested filter is too large")
	}
	rawBitCount := uint64(math.Ceil(m))
	bitCount := ((rawBitCount + 63) / 64) * 64
	probeFloat := math.Round(float64(bitCount) / float64(cfg.PlannedItems) * math.Ln2)
	if probeFloat < 1 {
		probeFloat = 1
	}
	// Reject configurations that would turn every lookup into unreasonable
	// CPU work in this educational implementation.
	if probeFloat > 64 {
		return 0, 0, fmt.Errorf("requested filter needs %.0f probes; maximum is 64", probeFloat)
	}
	return bitCount, uint64(probeFloat), nil
}

func modeledFalsePositiveRate(insertedItems, bitCount, probeCount uint64) float64 {
	setProbability := 1 - math.Exp(-float64(probeCount)*float64(insertedItems)/float64(bitCount))
	return math.Pow(setProbability, float64(probeCount))
}

// Record is one exact key/value entry in an immutable segment.
type Record struct {
	Key   string
	Value string
}

// Segment packages an exact sorted index and its optional filter in one
// immutable generation.
type Segment struct {
	records []Record
	filter  *Filter
}

// NewSegment copies and sorts records. A nil config disables filtering. Keys
// must be unique within the segment.
func NewSegment(records []Record, cfg *Config) (*Segment, error) {
	return newSegment(records, cfg, false)
}

func newSegment(records []Record, cfg *Config, deterministic bool) (*Segment, error) {
	ordered := slices.Clone(records)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Key < ordered[j].Key })
	for i := 1; i < len(ordered); i++ {
		if ordered[i-1].Key == ordered[i].Key {
			return nil, fmt.Errorf("duplicate segment key %q", ordered[i].Key)
		}
	}
	s := &Segment{records: ordered}
	if cfg == nil {
		return s, nil
	}
	keys := make([][]byte, len(ordered))
	for i := range ordered {
		keys[i] = []byte(ordered[i].Key)
	}
	var filter *Filter
	var err error
	if deterministic {
		filter, err = buildDeterministicMembership(keys, *cfg)
	} else {
		filter, err = BuildMembership(keys, *cfg)
	}
	if err != nil {
		return nil, err
	}
	s.filter = filter
	return s, nil
}

// FilterStats returns the segment's filter metadata and whether it has one.
func (s *Segment) FilterStats() (FilterStats, bool) {
	if s.filter == nil {
		return FilterStats{}, false
	}
	return s.filter.Stats(), true
}

// LookupStats counts work in the complete filter-plus-exact lookup path.
type LookupStats struct {
	CandidateSegments  uint64
	FilterQueries      uint64
	FilterProbes       uint64
	DefiniteNegatives  uint64
	PossibleMatches    uint64
	ExactChecks        uint64
	ExactComparisons   uint64
	TrueHits           uint64
	FalsePositives     uint64
	ExactChecksAvoided uint64
}

// Add accumulates another lookup's operation counts.
func (s *LookupStats) Add(other LookupStats) {
	s.CandidateSegments += other.CandidateSegments
	s.FilterQueries += other.FilterQueries
	s.FilterProbes += other.FilterProbes
	s.DefiniteNegatives += other.DefiniteNegatives
	s.PossibleMatches += other.PossibleMatches
	s.ExactChecks += other.ExactChecks
	s.ExactComparisons += other.ExactComparisons
	s.TrueHits += other.TrueHits
	s.FalsePositives += other.FalsePositives
	s.ExactChecksAvoided += other.ExactChecksAvoided
}

// LookupSegments searches segments in supplied order and stops at the first
// exact match. A Bloom-positive result never supplies the answer.
func LookupSegments(segments []*Segment, key string) (Record, bool, LookupStats) {
	var stats LookupStats
	for _, segment := range segments {
		stats.CandidateSegments++
		if segment.filter != nil {
			stats.FilterQueries++
			possible, probes := segment.filter.query([]byte(key))
			stats.FilterProbes += probes
			if !possible {
				stats.DefiniteNegatives++
				stats.ExactChecksAvoided++
				continue
			}
			stats.PossibleMatches++
		}

		stats.ExactChecks++
		record, found, comparisons := segment.lookupExact(key)
		stats.ExactComparisons += comparisons
		if found {
			stats.TrueHits++
			return record, true, stats
		}
		if segment.filter != nil {
			stats.FalsePositives++
		}
	}
	return Record{}, false, stats
}

func (s *Segment) lookupExact(key string) (Record, bool, uint64) {
	low, high := 0, len(s.records)
	var comparisons uint64
	for low < high {
		mid := int(uint(low+high) >> 1)
		comparisons++
		if s.records[mid].Key < key {
			low = mid + 1
		} else {
			high = mid
		}
	}
	if low < len(s.records) {
		comparisons++
		if s.records[low].Key == key {
			return s.records[low], true, comparisons
		}
	}
	return Record{}, false, comparisons
}
