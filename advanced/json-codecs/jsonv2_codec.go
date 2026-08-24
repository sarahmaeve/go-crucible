//go:build goexperiment.jsonv2

package jsoncodecs

import (
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"io"
)

type goJSONV2Codec struct{}

func (goJSONV2Codec) Name() string                  { return "go-json-v2" }
func (goJSONV2Codec) Marshal(v any) ([]byte, error) { return jsonv2.Marshal(v) }
func (goJSONV2Codec) Unmarshal(data []byte, v any) error {
	return jsonv2.Unmarshal(data, v)
}
func (goJSONV2Codec) NewEncoder(w io.Writer) StreamEncoder {
	return v2StreamEncoder{Encoder: jsontext.NewEncoder(w)}
}
func (goJSONV2Codec) NewDecoder(r io.Reader) StreamDecoder {
	return v2StreamDecoder{Decoder: jsontext.NewDecoder(r)}
}

type v2StreamEncoder struct {
	*jsontext.Encoder
}

func (e v2StreamEncoder) Encode(v any) error {
	return jsonv2.MarshalEncode(e.Encoder, v)
}

type v2StreamDecoder struct {
	*jsontext.Decoder
}

func (d v2StreamDecoder) Decode(v any) error {
	return jsonv2.UnmarshalDecode(d.Decoder, v)
}

// GoJSONV2 uses the direct Go 1.27 encoding/json/v2 API and its stricter
// defaults. It is intentionally separate from the encoding/json v1 facade.
var GoJSONV2 Codec = goJSONV2Codec{}

func init() {
	registeredCodecs = append(registeredCodecs, GoJSONV2)
}
