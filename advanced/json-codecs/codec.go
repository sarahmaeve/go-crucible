// Package jsoncodecs provides representative workloads for evaluating JSON
// codecs. It is a measurement lab, not an application codec abstraction.
package jsoncodecs

import (
	"encoding/json"
	"io"
	"reflect"

	"github.com/bytedance/sonic"
)

// Codec is the common surface used by the lab's correctness checks and
// benchmarks.
type Codec interface {
	Name() string
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, v any) error
	NewEncoder(w io.Writer) StreamEncoder
	NewDecoder(r io.Reader) StreamDecoder
}

// StreamEncoder is the common portion of encoding/json and Sonic encoders
// exercised by the lab.
type StreamEncoder interface {
	Encode(v any) error
}

// StreamDecoder is the common portion of encoding/json and Sonic decoders
// exercised by the lab.
type StreamDecoder interface {
	Decode(v any) error
}

type stdCodec struct{}

func (stdCodec) Name() string                       { return "encoding-json" }
func (stdCodec) Marshal(v any) ([]byte, error)      { return json.Marshal(v) }
func (stdCodec) Unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }
func (stdCodec) NewEncoder(w io.Writer) StreamEncoder {
	return json.NewEncoder(w)
}
func (stdCodec) NewDecoder(r io.Reader) StreamDecoder {
	return json.NewDecoder(r)
}

type sonicCodec struct {
	name string
	api  sonic.API
}

func (c sonicCodec) Name() string                       { return c.name }
func (c sonicCodec) Marshal(v any) ([]byte, error)      { return c.api.Marshal(v) }
func (c sonicCodec) Unmarshal(data []byte, v any) error { return c.api.Unmarshal(data, v) }
func (c sonicCodec) NewEncoder(w io.Writer) StreamEncoder {
	return c.api.NewEncoder(w)
}
func (c sonicCodec) NewDecoder(r io.Reader) StreamDecoder {
	return c.api.NewDecoder(r)
}

// StandardLibrary uses encoding/json.
var StandardLibrary Codec = stdCodec{}

// SonicStandard uses Sonic's compatibility-oriented configuration.
var SonicStandard Codec = sonicCodec{name: "sonic-std", api: sonic.ConfigStd}

// SonicDefault uses Sonic's default performance-oriented configuration. Its
// wire output is not promised to be byte-for-byte compatible with encoding/json.
var SonicDefault Codec = sonicCodec{name: "sonic-default", api: sonic.ConfigDefault}

var registeredCodecs = []Codec{StandardLibrary, SonicStandard, SonicDefault}

// Codecs returns every implementation used by the lab.
func Codecs() []Codec {
	return append([]Codec(nil), registeredCodecs...)
}

// CodecNamed returns one lab codec by its stable command-line name.
func CodecNamed(name string) (Codec, bool) {
	for _, codec := range Codecs() {
		if codec.Name() == name {
			return codec, true
		}
	}
	return nil, false
}

// StandardJSONBackend reports which implementation backs encoding/json in the
// current build. Go 1.27 temporarily permits selecting the legacy engine with
// GOEXPERIMENT=nojsonv2.
func StandardJSONBackend() string {
	return standardJSONBackend
}

// PretouchWorkloads compiles Sonic's encoder and decoder paths for the lab's
// representative types. Applications should pretouch only schemas they have
// measured and identified as latency-sensitive.
func PretouchWorkloads() error {
	return sonic.PretouchMany([]reflect.Type{
		reflect.TypeOf(HealthStatus{}),
		reflect.TypeOf(FlatBatch{}),
		reflect.TypeOf(MetricBatch{}),
		reflect.TypeOf(FindingList{}),
	})
}
