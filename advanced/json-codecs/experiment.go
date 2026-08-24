package jsoncodecs

import (
	"fmt"
	"io"
	"runtime"
	"text/tabwriter"
	"time"
)

// Measurement records one local elapsed-time observation. It is evidence for
// this process and workload, not a portable performance guarantee.
type Measurement struct {
	Workload     string
	Codec        string
	Operation    string
	PayloadBytes int
	Iterations   int
	First        time.Duration
	Steady       time.Duration
}

// NanosecondsPerOperation reports the average after the separately measured
// first operation.
func (m Measurement) NanosecondsPerOperation() float64 {
	if m.Iterations == 0 {
		return 0
	}
	return float64(m.Steady.Nanoseconds()) / float64(m.Iterations)
}

// MegabytesPerSecond reports decimal payload megabytes processed per second.
func (m Measurement) MegabytesPerSecond() float64 {
	if m.Steady <= 0 {
		return 0
	}
	return float64(m.PayloadBytes*m.Iterations) / m.Steady.Seconds() / 1_000_000
}

// RunExperiment executes the lab matrix. Call PretouchWorkloads first to
// compare explicitly warmed Sonic schemas with on-demand compilation.
func RunExperiment(iterations int) ([]Measurement, error) {
	return RunExperimentWithCodecs(iterations, Codecs())
}

// RunExperimentWithCodecs executes the lab matrix for a selected codec set.
// Selecting one Sonic codec in a fresh process avoids warming its shared type
// state through another Sonic configuration before the first measurement.
func RunExperimentWithCodecs(iterations int, codecs []Codec) ([]Measurement, error) {
	if iterations <= 0 {
		return nil, fmt.Errorf("iterations must be positive")
	}
	if len(codecs) == 0 {
		return nil, fmt.Errorf("at least one codec is required")
	}
	workloads, err := RepresentativeWorkloads()
	if err != nil {
		return nil, err
	}
	measurements := make([]Measurement, 0, len(workloads)*len(codecs)*2)
	for _, workload := range workloads {
		for _, codec := range codecs {
			measurement, err := measureMarshal(codec, workload, iterations)
			if err != nil {
				return nil, err
			}
			measurements = append(measurements, measurement)
			measurement, err = measureUnmarshal(codec, workload, iterations)
			if err != nil {
				return nil, err
			}
			measurements = append(measurements, measurement)
		}
	}
	return measurements, nil
}

func measureMarshal(codec Codec, workload Workload, iterations int) (Measurement, error) {
	start := time.Now()
	data, err := codec.Marshal(workload.Value)
	first := time.Since(start)
	if err != nil {
		return Measurement{}, fmt.Errorf("%s marshal %s: %w", codec.Name(), workload.Name, err)
	}
	start = time.Now()
	for i := 0; i < iterations; i++ {
		data, err = codec.Marshal(workload.Value)
		if err != nil {
			return Measurement{}, fmt.Errorf("%s marshal %s: %w", codec.Name(), workload.Name, err)
		}
	}
	runtime.KeepAlive(data)
	return Measurement{Workload: workload.Name, Codec: codec.Name(), Operation: "marshal", PayloadBytes: len(workload.JSON), Iterations: iterations, First: first, Steady: time.Since(start)}, nil
}

func measureUnmarshal(codec Codec, workload Workload, iterations int) (Measurement, error) {
	value := workload.New()
	start := time.Now()
	err := codec.Unmarshal(workload.JSON, value)
	first := time.Since(start)
	if err != nil {
		return Measurement{}, fmt.Errorf("%s unmarshal %s: %w", codec.Name(), workload.Name, err)
	}
	start = time.Now()
	for i := 0; i < iterations; i++ {
		value = workload.New()
		if err := codec.Unmarshal(workload.JSON, value); err != nil {
			return Measurement{}, fmt.Errorf("%s unmarshal %s: %w", codec.Name(), workload.Name, err)
		}
	}
	runtime.KeepAlive(value)
	return Measurement{Workload: workload.Name, Codec: codec.Name(), Operation: "unmarshal", PayloadBytes: len(workload.JSON), Iterations: iterations, First: first, Steady: time.Since(start)}, nil
}

// WriteMeasurements prints a compact comparison table.
func WriteMeasurements(w io.Writer, measurements []Measurement) error {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "WORKLOAD\tBYTES\tOPERATION\tCODEC\tFIRST\tSTEADY NS/OP\tMB/S"); err != nil {
		return err
	}
	for _, measurement := range measurements {
		if _, err := fmt.Fprintf(tw, "%s\t%d\t%s\t%s\t%s\t%.0f\t%.1f\n",
			measurement.Workload,
			measurement.PayloadBytes,
			measurement.Operation,
			measurement.Codec,
			measurement.First,
			measurement.NanosecondsPerOperation(),
			measurement.MegabytesPerSecond(),
		); err != nil {
			return err
		}
	}
	return tw.Flush()
}
