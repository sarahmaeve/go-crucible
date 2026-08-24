package jsoncodecs

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunExperimentShape(t *testing.T) {
	measurements, err := RunExperiment(1)
	if err != nil {
		t.Fatal(err)
	}
	workloads, err := RepresentativeWorkloads()
	if err != nil {
		t.Fatal(err)
	}
	want := len(Codecs()) * len(workloads) * 2
	if len(measurements) != want {
		t.Fatalf("measurements = %d, want %d", len(measurements), want)
	}
	for _, measurement := range measurements {
		if measurement.PayloadBytes <= 0 || measurement.First <= 0 || measurement.Steady <= 0 {
			t.Fatalf("incomplete measurement: %+v", measurement)
		}
	}
}

func TestRunExperimentWithCodecsRequiresSelection(t *testing.T) {
	if _, err := RunExperimentWithCodecs(1, nil); err == nil {
		t.Fatal("empty codec selection succeeded")
	}
}

func TestWriteMeasurements(t *testing.T) {
	measurements, err := RunExperiment(1)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := WriteMeasurements(&output, measurements); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"WORKLOAD", "small-health", "encoding-json", "sonic-std", "sonic-default"} {
		if !strings.Contains(output.String(), text) {
			t.Fatalf("output does not contain %q:\n%s", text, output.String())
		}
	}
}
