package jsoncodecs

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkHTTPMetricIngestion(b *testing.B) {
	workloads, err := RepresentativeWorkloads()
	if err != nil {
		b.Fatal(err)
	}
	workload, ok := findWorkload(workloads, "medium-metrics")
	if !ok {
		b.Fatal("medium-metrics workload is missing")
	}
	body := workload.JSON
	for _, codec := range Codecs() {
		b.Run(codec.Name(), func(b *testing.B) {
			handler := metricHandler(codec)
			b.ReportAllocs()
			b.SetBytes(int64(len(body)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				req := httptest.NewRequest(http.MethodPost, "/v1/metrics", bytes.NewReader(body))
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)
				if rec.Code != http.StatusAccepted {
					b.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
				}
			}
		})
	}
}

func metricHandler(codec Codec) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		defer r.Body.Close()
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var batch MetricBatch
		if err := codec.Unmarshal(data, &batch); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		response, err := codec.Marshal(HealthStatus{Status: "accepted"})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write(response)
	})
}
