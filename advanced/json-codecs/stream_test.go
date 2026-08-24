package jsoncodecs

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

var benchmarkMetric Metric

func TestStreamingMetricsRoundTrip(t *testing.T) {
	metrics := makeMetricBatch(32).Metrics
	for _, codec := range Codecs() {
		t.Run(codec.Name(), func(t *testing.T) {
			var stream bytes.Buffer
			encoder := codec.NewEncoder(&stream)
			for index := range metrics {
				if err := encoder.Encode(&metrics[index]); err != nil {
					t.Fatalf("encode item %d: %v", index, err)
				}
			}

			decoder := codec.NewDecoder(bytes.NewReader(stream.Bytes()))
			decoded := make([]Metric, 0, len(metrics))
			for {
				var metric Metric
				err := decoder.Decode(&metric)
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("decode item %d: %v", len(decoded), err)
				}
				decoded = append(decoded, metric)
			}
			if !reflect.DeepEqual(decoded, metrics) {
				t.Fatal("stream round trip changed metrics")
			}
		})
	}
}

func BenchmarkStreamEncodeMetrics(b *testing.B) {
	metrics := makeMetricBatch(128).Metrics
	baseline, err := encodeMetricStream(StandardLibrary, metrics)
	if err != nil {
		b.Fatal(err)
	}
	for _, codec := range Codecs() {
		b.Run(codec.Name(), func(b *testing.B) {
			var stream bytes.Buffer
			stream.Grow(len(baseline))
			b.ReportAllocs()
			b.SetBytes(int64(len(baseline)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				stream.Reset()
				encoder := codec.NewEncoder(&stream)
				for index := range metrics {
					if err := encoder.Encode(&metrics[index]); err != nil {
						b.Fatal(err)
					}
				}
			}
			benchmarkBytes = stream.Bytes()
		})
	}
}

func BenchmarkStreamDecodeMetrics(b *testing.B) {
	metrics := makeMetricBatch(128).Metrics
	baseline, err := encodeMetricStream(StandardLibrary, metrics)
	if err != nil {
		b.Fatal(err)
	}
	for _, codec := range Codecs() {
		b.Run(codec.Name(), func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(baseline)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				decoder := codec.NewDecoder(bytes.NewReader(baseline))
				for range metrics {
					var metric Metric
					if err := decoder.Decode(&metric); err != nil {
						b.Fatal(err)
					}
					benchmarkMetric = metric
				}
			}
		})
	}
}

func encodeMetricStream(codec Codec, metrics []Metric) ([]byte, error) {
	var stream bytes.Buffer
	encoder := codec.NewEncoder(&stream)
	for index := range metrics {
		if err := encoder.Encode(&metrics[index]); err != nil {
			return nil, err
		}
	}
	return stream.Bytes(), nil
}
