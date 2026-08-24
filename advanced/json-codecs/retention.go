package jsoncodecs

import (
	"bytes"
	"fmt"
	"runtime"
	"runtime/debug"
)

type retentionEnvelope struct {
	Keep string `json:"keep"`
}

// RetentionResult records live-heap growth while small decoded strings remain
// reachable after their much larger source documents have otherwise died.
// Heap measurements are noisy and must be compared in fresh processes.
type RetentionResult struct {
	Codec            string
	Documents        int
	PaddingPerInput  int
	LogicalKeepBytes int
	HeapGrowthBytes  uint64
}

// RunRetentionExperiment decodes documents with one retained short field and
// one ignored large field. It warms the codec before the baseline so JIT state
// is not mistaken for retained input storage.
func RunRetentionExperiment(codec Codec, documents, paddingBytes int) (RetentionResult, error) {
	if codec == nil {
		return RetentionResult{}, fmt.Errorf("codec must not be nil")
	}
	if documents <= 0 {
		return RetentionResult{}, fmt.Errorf("documents must be positive")
	}
	if paddingBytes < 0 {
		return RetentionResult{}, fmt.Errorf("padding bytes must not be negative")
	}

	var warm retentionEnvelope
	if err := codec.Unmarshal(retentionDocument(0, 128), &warm); err != nil {
		return RetentionResult{}, fmt.Errorf("warm %s: %w", codec.Name(), err)
	}
	debug.FreeOSMemory()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	retained := make([]string, 0, documents)
	logicalBytes := 0
	for index := 0; index < documents; index++ {
		data := retentionDocument(index, paddingBytes)
		var envelope retentionEnvelope
		if err := codec.Unmarshal(data, &envelope); err != nil {
			return RetentionResult{}, fmt.Errorf("decode document %d with %s: %w", index, codec.Name(), err)
		}
		retained = append(retained, envelope.Keep)
		logicalBytes += len(envelope.Keep)
	}

	debug.FreeOSMemory()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	runtime.KeepAlive(retained)
	growth := uint64(0)
	if after.HeapAlloc > before.HeapAlloc {
		growth = after.HeapAlloc - before.HeapAlloc
	}
	return RetentionResult{
		Codec:            codec.Name(),
		Documents:        documents,
		PaddingPerInput:  paddingBytes,
		LogicalKeepBytes: logicalBytes,
		HeapGrowthBytes:  growth,
	}, nil
}

func retentionDocument(index, paddingBytes int) []byte {
	var document bytes.Buffer
	document.Grow(paddingBytes + 64)
	fmt.Fprintf(&document, `{"keep":"token-%08d","padding":"`, index)
	document.Write(bytes.Repeat([]byte{'x'}, paddingBytes))
	document.WriteString(`"}`)
	return document.Bytes()
}
