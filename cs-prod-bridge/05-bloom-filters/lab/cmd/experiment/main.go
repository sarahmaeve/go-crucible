package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"text/tabwriter"

	"github.com/go-crucible/go-crucible/cs-prod-bridge/05-bloom-filters/lab"
)

func main() {
	custom := flag.Bool("custom", false, "run one custom workload instead of the standard matrix")
	segments := flag.Int("segments", 8, "immutable segments per query")
	keys := flag.Int("keys", 200, "actual keys in each segment")
	planned := flag.Uint64("planned", 200, "planned keys used to size each filter")
	target := flag.Float64("target", 0.01, "target false-positive probability")
	maxBytes := flag.Uint64("max-bytes", 0, "maximum bytes for each filter; zero uses the lab default")
	queries := flag.Int("queries", 1000, "queries in the workload")
	absent := flag.Float64("absent", 1, "fraction of queries that are absent, from 0 to 1")
	repeat := flag.Bool("repeat-absent", false, "repeat one engineered false-positive absent key")
	exactCost := flag.Uint64("exact-cost", 100, "modeled cost units for one exact check")
	flag.Parse()

	var err error
	if *custom {
		err = writeCustom(os.Stdout, lab.ExperimentSpec{
			Name:                    "custom",
			SegmentCount:            *segments,
			KeysPerSegment:          *keys,
			PlannedItemsPerSegment:  *planned,
			TargetFalsePositiveRate: *target,
			MaxBytesPerSegment:      *maxBytes,
			QueryCount:              *queries,
			AbsentFraction:          *absent,
			RepeatOneAbsentKey:      *repeat,
		}, *exactCost)
	} else {
		err = lab.WriteExperimentTable(os.Stdout)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func writeCustom(w io.Writer, spec lab.ExperimentSpec, exactCost uint64) error {
	result, err := lab.RunExperiment(spec)
	if err != nil {
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	rows := []struct {
		label string
		value any
	}{
		{"candidate segments", result.Stats.CandidateSegments},
		{"filter queries", result.Stats.FilterQueries},
		{"filter probes", result.Stats.FilterProbes},
		{"exact checks", result.Stats.ExactChecks},
		{"exact checks avoided", result.Stats.ExactChecksAvoided},
		{"true hits", result.Stats.TrueHits},
		{"false positives", result.Stats.FalsePositives},
		{"filter bytes", result.FilterBytes},
		{"maximum bytes per filter", result.MaximumFilterBytes},
		{"bit density", fmt.Sprintf("%.3f", result.BitDensity)},
		{"modeled false-positive rate", fmt.Sprintf("%.4g", result.ModeledFalsePositiveRate)},
		{"observed false-positive rate", fmt.Sprintf("%.4g", result.ObservedFalsePositiveRate())},
		{"absent queries", result.AbsentQueries},
		{"distinct absent keys", result.DistinctAbsentKeys},
		{"filter-path cost", result.FilterCost(exactCost)},
		{"exact-only cost", result.ExactOnlyCost(exactCost)},
	}
	for _, row := range rows {
		if _, err := fmt.Fprintf(tw, "%s\t%v\n", row.label, row.value); err != nil {
			return err
		}
	}
	if result.RepeatedKey != "" {
		if _, err := fmt.Fprintf(tw, "repeated key\t%s\n", result.RepeatedKey); err != nil {
			return err
		}
	}
	return tw.Flush()
}
