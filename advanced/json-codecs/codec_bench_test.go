package jsoncodecs

import (
	"sync"
	"testing"
)

var (
	benchmarkBytes []byte
	benchmarkValue any
)

func BenchmarkMarshal(b *testing.B) {
	workloads, err := RepresentativeWorkloads()
	if err != nil {
		b.Fatal(err)
	}
	for _, workload := range workloads {
		for _, codec := range Codecs() {
			b.Run(workload.Name+"/"+codec.Name(), func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(workload.JSON)))
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					data, marshalErr := codec.Marshal(workload.Value)
					if marshalErr != nil {
						b.Fatal(marshalErr)
					}
					benchmarkBytes = data
				}
			})
		}
	}
}

func BenchmarkUnmarshal(b *testing.B) {
	workloads, err := RepresentativeWorkloads()
	if err != nil {
		b.Fatal(err)
	}
	for _, workload := range workloads {
		for _, codec := range Codecs() {
			b.Run(workload.Name+"/"+codec.Name(), func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(workload.JSON)))
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					value := workload.New()
					if err := codec.Unmarshal(workload.JSON, value); err != nil {
						b.Fatal(err)
					}
					benchmarkValue = value
				}
			})
		}
	}
}

func BenchmarkParallelMediumUnmarshal(b *testing.B) {
	workloads, err := RepresentativeWorkloads()
	if err != nil {
		b.Fatal(err)
	}
	workload, ok := findWorkload(workloads, "medium-metrics")
	if !ok {
		b.Fatal("medium-metrics workload is missing")
	}
	for _, codec := range Codecs() {
		b.Run(codec.Name(), func(b *testing.B) {
			var (
				decodeErr error
				errOnce   sync.Once
			)
			b.ReportAllocs()
			b.SetBytes(int64(len(workload.JSON)))
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					value := workload.New()
					if err := codec.Unmarshal(workload.JSON, value); err != nil {
						errOnce.Do(func() { decodeErr = err })
						return
					}
				}
			})
			if decodeErr != nil {
				b.Fatal(decodeErr)
			}
		})
	}
}
