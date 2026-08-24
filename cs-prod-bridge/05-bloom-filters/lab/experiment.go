package lab

import (
	"fmt"
	"io"
	"math"
	"text/tabwriter"
)

// ExperimentSpec describes one repeatable segment workload. Hits are selected
// from the final segment so every query considers the same number of segments.
type ExperimentSpec struct {
	Name                    string
	SegmentCount            int
	KeysPerSegment          int
	PlannedItemsPerSegment  uint64
	TargetFalsePositiveRate float64
	MaxBytesPerSegment      uint64
	QueryCount              int
	AbsentFraction          float64
	RepeatOneAbsentKey      bool
}

// ExperimentResult contains observable operation counts and filter state.
type ExperimentResult struct {
	Spec                     ExperimentSpec
	Stats                    LookupStats
	FilterBytes              uint64
	MaximumFilterBytes       uint64
	BitDensity               float64
	ModeledFalsePositiveRate float64
	AbsentQueries            uint64
	DistinctAbsentKeys       uint64
	RepeatedKey              string
}

// FilterCost estimates work with one unit per inspected bit position and
// exactCost units per exact check. It is a comparison model, not elapsed time.
func (r ExperimentResult) FilterCost(exactCost uint64) uint64 {
	return r.Stats.FilterProbes + r.Stats.ExactChecks*exactCost
}

// ExactOnlyCost estimates the same workload with no filters.
func (r ExperimentResult) ExactOnlyCost(exactCost uint64) uint64 {
	return r.Stats.CandidateSegments * exactCost
}

// ObservedFalsePositiveRate reports exact misses after possible matches divided
// by all exact-source nonmembers queried. It returns zero when there were none.
func (r ExperimentResult) ObservedFalsePositiveRate() float64 {
	nonmembers := r.Stats.DefiniteNegatives + r.Stats.FalsePositives
	if nonmembers == 0 {
		return 0
	}
	return float64(r.Stats.FalsePositives) / float64(nonmembers)
}

// RunExperiment builds deterministic educational filters and executes a
// workload. Unlike BuildMembership, its stable hash is intentionally fixed so
// learners can reproduce the table. It is not a supported persistent format.
func RunExperiment(spec ExperimentSpec) (ExperimentResult, error) {
	if spec.Name == "" {
		return ExperimentResult{}, fmt.Errorf("experiment name must not be empty")
	}
	if spec.SegmentCount <= 0 || spec.KeysPerSegment <= 0 || spec.QueryCount <= 0 {
		return ExperimentResult{}, fmt.Errorf("segment, key, and query counts must be positive")
	}
	if spec.PlannedItemsPerSegment == 0 {
		return ExperimentResult{}, fmt.Errorf("planned items per segment must be positive")
	}
	if math.IsNaN(spec.AbsentFraction) || spec.AbsentFraction < 0 || spec.AbsentFraction > 1 {
		return ExperimentResult{}, fmt.Errorf("absent fraction must be between 0 and 1")
	}

	segments := make([]*Segment, spec.SegmentCount)
	var totalSetBits, totalBits uint64
	for segmentIndex := range segments {
		records := experimentRecords(segmentIndex, spec.KeysPerSegment)
		cfg := Config{
			PlannedItems:            spec.PlannedItemsPerSegment,
			TargetFalsePositiveRate: spec.TargetFalsePositiveRate,
			MaxBytes:                spec.MaxBytesPerSegment,
		}
		segment, err := newSegment(records, &cfg, true)
		if err != nil {
			return ExperimentResult{}, fmt.Errorf("build segment %d: %w", segmentIndex, err)
		}
		segments[segmentIndex] = segment
		stats := segment.filter.Stats()
		totalSetBits += stats.SetBits
		totalBits += stats.BitCount
	}

	result := ExperimentResult{Spec: spec}
	result.FilterBytes = totalBits / 8
	result.MaximumFilterBytes = segments[0].filter.Stats().MaximumBytes
	result.BitDensity = float64(totalSetBits) / float64(totalBits)
	// Every segment has identical dimensions and cardinality in one experiment,
	// so one segment's modeled rate describes the matrix row.
	result.ModeledFalsePositiveRate = segments[0].filter.Stats().ModeledFalsePositiveRate
	absentCount := int(math.Round(float64(spec.QueryCount) * spec.AbsentFraction))
	result.AbsentQueries = uint64(absentCount)

	var repeatedKey string
	if spec.RepeatOneAbsentKey && absentCount > 0 {
		var err error
		repeatedKey, err = findFalsePositive(segments)
		if err != nil {
			return ExperimentResult{}, err
		}
		result.RepeatedKey = repeatedKey
		result.DistinctAbsentKeys = 1
	} else {
		result.DistinctAbsentKeys = uint64(absentCount)
	}

	for queryIndex := 0; queryIndex < spec.QueryCount; queryIndex++ {
		var key string
		if queryIndex < absentCount {
			key = repeatedKey
			if key == "" {
				key = fmt.Sprintf("absent-%06d", queryIndex)
			}
		} else {
			key = fmt.Sprintf("segment-%02d-key-%06d", spec.SegmentCount-1, queryIndex%spec.KeysPerSegment)
		}
		_, _, stats := LookupSegments(segments, key)
		result.Stats.Add(stats)
	}
	return result, nil
}

// WriteExperimentTable prints the lab's standard experiment matrix.
func WriteExperimentTable(w io.Writer) error {
	specs := standardExperiments()
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "SCENARIO\tABSENT\tTARGET\tMODELED-P\tOBS-P\tLOAD\tDENSITY\tBYTES\tUNIQUE-ABS\tPROBES\tEXACT\tAVOIDED\tFP\tCOST Ce=1\tCOST Ce=100"); err != nil {
		return err
	}
	for _, spec := range specs {
		result, err := RunExperiment(spec)
		if err != nil {
			return fmt.Errorf("run %s: %w", spec.Name, err)
		}
		load := float64(spec.KeysPerSegment) / float64(spec.PlannedItemsPerSegment)
		if _, err := fmt.Fprintf(tw, "%s\t%.0f%%\t%.1g\t%.2g\t%.2g\t%.0fx\t%.3f\t%d\t%d\t%d\t%d\t%d\t%d\t%d/%d\t%d/%d\n",
			spec.Name,
			spec.AbsentFraction*100,
			spec.TargetFalsePositiveRate,
			result.ModeledFalsePositiveRate,
			result.ObservedFalsePositiveRate(),
			load,
			result.BitDensity,
			result.FilterBytes,
			result.DistinctAbsentKeys,
			result.Stats.FilterProbes,
			result.Stats.ExactChecks,
			result.Stats.ExactChecksAvoided,
			result.Stats.FalsePositives,
			result.FilterCost(1), result.ExactOnlyCost(1),
			result.FilterCost(100), result.ExactOnlyCost(100),
		); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(tw, "\nCOST cells are filter-path/exact-only work units; inspected bit=1 unit."); err != nil {
		return err
	}
	return tw.Flush()
}

func standardExperiments() []ExperimentSpec {
	base := ExperimentSpec{
		SegmentCount:            8,
		KeysPerSegment:          200,
		PlannedItemsPerSegment:  200,
		TargetFalsePositiveRate: 0.01,
		QueryCount:              1_000,
	}
	makeSpec := func(name string, mutate func(*ExperimentSpec)) ExperimentSpec {
		spec := base
		spec.Name = name
		mutate(&spec)
		return spec
	}
	return []ExperimentSpec{
		makeSpec("single-segment-all-hits", func(s *ExperimentSpec) { s.SegmentCount = 1; s.AbsentFraction = 0 }),
		makeSpec("mix-0%-absent", func(s *ExperimentSpec) { s.AbsentFraction = 0 }),
		makeSpec("mix-50%-absent", func(s *ExperimentSpec) { s.AbsentFraction = 0.5 }),
		makeSpec("mix-90%-absent", func(s *ExperimentSpec) { s.AbsentFraction = 0.9 }),
		makeSpec("mix-100%-absent", func(s *ExperimentSpec) { s.AbsentFraction = 1 }),
		makeSpec("target-10%", func(s *ExperimentSpec) { s.AbsentFraction = 1; s.TargetFalsePositiveRate = 0.1 }),
		makeSpec("target-0.1%", func(s *ExperimentSpec) { s.AbsentFraction = 1; s.TargetFalsePositiveRate = 0.001 }),
		makeSpec("capacity-2x", func(s *ExperimentSpec) { s.AbsentFraction = 1; s.PlannedItemsPerSegment = 100 }),
		makeSpec("capacity-10x", func(s *ExperimentSpec) { s.AbsentFraction = 1; s.PlannedItemsPerSegment = 20 }),
		makeSpec("retry-uniform", func(s *ExperimentSpec) { s.AbsentFraction = 1 }),
		makeSpec("retry-one-hot-key", func(s *ExperimentSpec) { s.AbsentFraction = 1; s.RepeatOneAbsentKey = true }),
	}
}

func findFalsePositive(segments []*Segment) (string, error) {
	for i := 0; i < 1_000_000; i++ {
		key := fmt.Sprintf("hot-absent-%06d", i)
		_, found, stats := LookupSegments(segments, key)
		if !found && stats.FalsePositives > 0 {
			return key, nil
		}
	}
	return "", fmt.Errorf("could not engineer a repeated false-positive key")
}

func experimentRecords(segmentIndex, count int) []Record {
	records := make([]Record, count)
	for keyIndex := range records {
		records[keyIndex] = Record{
			Key:   fmt.Sprintf("segment-%02d-key-%06d", segmentIndex, keyIndex),
			Value: fmt.Sprintf("value-%02d-%06d", segmentIndex, keyIndex),
		}
	}
	return records
}

func stableHashPair(key []byte) (uint64, uint64) {
	return stableHash64(key, 14695981039346656037), stableHash64(key, 7809847782465536322)
}

func stableHash64(key []byte, initial uint64) uint64 {
	hash := initial
	for _, value := range key {
		hash ^= uint64(value)
		hash *= 1099511628211
	}
	// Avalanche the FNV state so nearby formatted keys do not retain visible
	// structure in their low bits.
	hash ^= hash >> 33
	hash *= 0xff51afd7ed558ccd
	hash ^= hash >> 33
	hash *= 0xc4ceb9fe1a85ec53
	hash ^= hash >> 33
	return hash
}
