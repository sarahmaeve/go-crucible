package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	jsoncodecs "github.com/go-crucible/go-crucible/advanced/json-codecs"
)

func main() {
	iterations := flag.Int("iterations", 100, "steady-state operations per codec and workload")
	pretouch := flag.Bool("pretouch", false, "compile Sonic workload schemas before measuring")
	codecName := flag.String("codec", "all", "codec to measure: all, encoding-json, go-json-v2, sonic-std, or sonic-default")
	flag.Parse()
	codecs, err := selectCodecs(*codecName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	if *pretouch {
		if err := jsoncodecs.PretouchWorkloads(); err != nil {
			fmt.Fprintf(os.Stderr, "pretouch workloads: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("go=%s os=%s arch=%s gomaxprocs=%d std-json=%s pretouch=%t iterations=%d\n\n",
		runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.GOMAXPROCS(0), jsoncodecs.StandardJSONBackend(), *pretouch, *iterations)
	measurements, err := jsoncodecs.RunExperimentWithCodecs(*iterations, codecs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "run experiment: %v\n", err)
		os.Exit(1)
	}
	if err := jsoncodecs.WriteMeasurements(os.Stdout, measurements); err != nil {
		fmt.Fprintf(os.Stderr, "write results: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("\nThese are local observations, not portable performance guarantees. Use go test -bench with benchstat for decisions.")
}

func selectCodecs(name string) ([]jsoncodecs.Codec, error) {
	if name == "all" {
		return jsoncodecs.Codecs(), nil
	}
	if codec, ok := jsoncodecs.CodecNamed(name); ok {
		return []jsoncodecs.Codec{codec}, nil
	}
	return nil, fmt.Errorf("unknown codec %q", name)
}
