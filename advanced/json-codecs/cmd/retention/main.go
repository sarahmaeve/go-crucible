package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	jsoncodecs "github.com/go-crucible/go-crucible/advanced/json-codecs"
)

func main() {
	codecName := flag.String("codec", "", "one codec: encoding-json, go-json-v2, sonic-std, or sonic-default")
	documents := flag.Int("documents", 16, "number of independently allocated input documents")
	paddingBytes := flag.Int("padding-bytes", 1<<20, "ignored padding bytes in each input")
	flag.Parse()

	codec, ok := jsoncodecs.CodecNamed(*codecName)
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown or missing codec %q; run each codec in a fresh process\n", *codecName)
		os.Exit(2)
	}
	result, err := jsoncodecs.RunRetentionExperiment(codec, *documents, *paddingBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "run retention experiment: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("go=%s os=%s arch=%s std-json=%s codec=%s documents=%d padding/input=%d\n",
		runtime.Version(), runtime.GOOS, runtime.GOARCH, jsoncodecs.StandardJSONBackend(), result.Codec, result.Documents, result.PaddingPerInput)
	fmt.Printf("logical kept string bytes: %d\n", result.LogicalKeepBytes)
	fmt.Printf("observed live heap growth:  %d bytes (%.2f MiB)\n", result.HeapGrowthBytes, float64(result.HeapGrowthBytes)/(1<<20))
	fmt.Println("Heap deltas are noisy. Repeat in fresh processes and compare directionally; do not assert an exact value.")
}
