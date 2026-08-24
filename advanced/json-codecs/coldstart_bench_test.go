package jsoncodecs

import (
	"io"
	"os"
	"os/exec"
	"testing"
)

const coldStartHelperCodec = "GO_CRUCIBLE_JSON_COLD_CODEC"

// TestColdStartHelper is entered only by BenchmarkColdProcess. Keeping the
// operation in a new test process prevents another codec configuration from
// warming Sonic's process-global type cache first.
func TestColdStartHelper(t *testing.T) {
	codecName := os.Getenv(coldStartHelperCodec)
	if codecName == "" {
		t.Skip("cold-start subprocess helper")
	}
	codec, ok := CodecNamed(codecName)
	if !ok {
		t.Fatalf("unknown helper codec %q", codecName)
	}
	workloads, err := RepresentativeWorkloads()
	if err != nil {
		t.Fatal(err)
	}
	workload, ok := findWorkload(workloads, "medium-metrics")
	if !ok {
		t.Fatal("medium-metrics workload is missing")
	}
	encoded, err := codec.Marshal(workload.Value)
	if err != nil {
		t.Fatal(err)
	}
	decoded := workload.New()
	if err := codec.Unmarshal(encoded, decoded); err != nil {
		t.Fatal(err)
	}
}

// BenchmarkColdProcess measures process start through one completed medium
// marshal/unmarshal operation. It includes Go test process startup and fixture
// construction for every codec; compare relative results on the same machine.
func BenchmarkColdProcess(b *testing.B) {
	executable, err := os.Executable()
	if err != nil {
		b.Fatal(err)
	}
	for _, codec := range Codecs() {
		b.Run(codec.Name(), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				cmd := exec.Command(executable, "-test.run=^TestColdStartHelper$", "-test.count=1")
				cmd.Env = append(os.Environ(), coldStartHelperCodec+"="+codec.Name())
				cmd.Stdout = io.Discard
				cmd.Stderr = io.Discard
				if err := cmd.Run(); err != nil {
					b.Fatalf("cold helper: %v", err)
				}
			}
		})
	}
}
