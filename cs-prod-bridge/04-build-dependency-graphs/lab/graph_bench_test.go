package lab

import (
	"fmt"
	"testing"
)

var (
	benchmarkOrder []TargetID
	benchmarkPath  []TargetID
)

func BenchmarkGraphQueries(b *testing.B) {
	for _, size := range []int{100, 1_000, 10_000} {
		b.Run(fmt.Sprintf("chain/v-%d/e-%d", size, size-1), func(b *testing.B) {
			g, root, leaf := benchmarkChain(b, size)
			b.Run("build-order", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					var err error
					benchmarkOrder, _, err = g.BuildOrder([]TargetID{root})
					if err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("fewest-edge-path", func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					var found bool
					var err error
					benchmarkPath, found, _, err = g.FewestEdgePath(root, leaf)
					if err != nil || !found {
						b.Fatalf("path error=%v found=%t", err, found)
					}
				}
			})
		})
	}

	for _, width := range []int{100, 1_000, 10_000} {
		b.Run(fmt.Sprintf("fanout/v-%d/e-%d", width+1, width), func(b *testing.B) {
			g, root := benchmarkFanout(b, width)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var err error
				benchmarkOrder, _, err = g.BuildOrder([]TargetID{root})
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func benchmarkChain(b *testing.B, size int) (*Graph, TargetID, TargetID) {
	b.Helper()
	targets := make([]TargetID, size)
	dependencies := make([]Dependency, 0, size-1)
	for i := range targets {
		targets[i] = TargetID(fmt.Sprintf("//chain:t%06d", i))
		if i > 0 {
			dependencies = append(dependencies, Dependency{Target: targets[i-1], Needs: targets[i]})
		}
	}
	g, err := NewGraph(targets, dependencies)
	if err != nil {
		b.Fatal(err)
	}
	return g, targets[0], targets[len(targets)-1]
}

func benchmarkFanout(b *testing.B, width int) (*Graph, TargetID) {
	b.Helper()
	root := TargetID("//fanout:root")
	targets := make([]TargetID, 1, width+1)
	targets[0] = root
	dependencies := make([]Dependency, 0, width)
	for i := 0; i < width; i++ {
		leaf := TargetID(fmt.Sprintf("//fanout:leaf-%06d", i))
		targets = append(targets, leaf)
		dependencies = append(dependencies, Dependency{Target: root, Needs: leaf})
	}
	g, err := NewGraph(targets, dependencies)
	if err != nil {
		b.Fatal(err)
	}
	return g, root
}
