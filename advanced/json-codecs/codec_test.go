package jsoncodecs

import (
	"bytes"
	"encoding/json"
	"math"
	"reflect"
	"testing"
)

func TestRepresentativeWorkloadsRoundTrip(t *testing.T) {
	workloads, err := RepresentativeWorkloads()
	if err != nil {
		t.Fatal(err)
	}
	for _, codec := range Codecs() {
		for _, workload := range workloads {
			t.Run(codec.Name()+"/"+workload.Name, func(t *testing.T) {
				encoded, err := codec.Marshal(workload.Value)
				if err != nil {
					t.Fatalf("marshal: %v", err)
				}
				decoded := workload.New()
				if err := codec.Unmarshal(encoded, decoded); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				want := reflect.ValueOf(workload.Value)
				got := reflect.ValueOf(decoded).Elem()
				if !reflect.DeepEqual(got.Interface(), want.Interface()) {
					t.Fatal("round trip changed the value")
				}
			})
		}
	}
}

func TestSonicStandardSelectedCompatibility(t *testing.T) {
	type payload struct {
		HTML  string         `json:"html"`
		Nil   []string       `json:"nil"`
		Empty []string       `json:"empty"`
		Map   map[string]int `json:"map"`
	}
	cases := []any{
		payload{HTML: "<script>&", Empty: []string{}, Map: map[string]int{"z": 1, "a": 2}},
		map[string]any{"finite": 1.25, "text": "line\nfeed"},
	}
	for _, value := range cases {
		want, err := StandardLibrary.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		got, err := SonicStandard.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("Sonic ConfigStd wire output differs:\nstd:   %s\nsonic: %s", want, got)
		}
	}

	for _, input := range [][]byte{
		[]byte(`{"HTML":"case folded"}`),
		[]byte(`{"HTML":"first","HTML":"last"}`),
		[]byte(`{"HTML":"ok"} trailing`),
		[]byte{'"', 0xff, '"'},
	} {
		var stdValue, sonicValue any
		stdErr := StandardLibrary.Unmarshal(input, &stdValue)
		sonicErr := SonicStandard.Unmarshal(input, &sonicValue)
		if (stdErr == nil) != (sonicErr == nil) {
			t.Fatalf("error presence differs for %q: std=%v sonic=%v", input, stdErr, sonicErr)
		}
		if stdErr == nil && !reflect.DeepEqual(stdValue, sonicValue) {
			t.Fatalf("decoded values differ for %q: std=%#v sonic=%#v", input, stdValue, sonicValue)
		}
	}
}

func TestConfigurationsExposeDocumentedOutputBehavior(t *testing.T) {
	value := map[string]string{"html": "<script>&"}
	std, err := StandardLibrary.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	compatible, err := SonicStandard.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	fast, err := SonicDefault.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(std, compatible) {
		t.Fatalf("ConfigStd = %s, want %s", compatible, std)
	}
	wantDefault := []byte(`{"html":"<script>&"}`)
	if bytes.Equal(std, fast) {
		t.Log("ConfigDefault matched encoding/json; Sonic documents this fallback on unsupported optimized environments")
	} else if !bytes.Equal(fast, wantDefault) {
		t.Fatalf("ConfigDefault = %s, want optimized output %s or fallback output %s", fast, wantDefault, std)
	}
}

func TestAllCodecsRejectUnsupportedFloat(t *testing.T) {
	for _, codec := range Codecs() {
		t.Run(codec.Name(), func(t *testing.T) {
			if _, err := codec.Marshal(math.Inf(1)); err == nil {
				t.Fatal("marshal infinity succeeded")
			}
		})
	}
}

func TestBaselineJSONIsValid(t *testing.T) {
	workloads, err := RepresentativeWorkloads()
	if err != nil {
		t.Fatal(err)
	}
	for _, workload := range workloads {
		if !json.Valid(workload.JSON) {
			t.Fatalf("%s fixture is invalid JSON", workload.Name)
		}
	}
}

func TestCodecNamed(t *testing.T) {
	for _, codec := range Codecs() {
		got, ok := CodecNamed(codec.Name())
		if !ok || got.Name() != codec.Name() {
			t.Fatalf("CodecNamed(%q) = %v, %t", codec.Name(), got, ok)
		}
	}
	if codec, ok := CodecNamed("not-a-codec"); ok || codec != nil {
		t.Fatalf("unknown codec = %v, %t", codec, ok)
	}
}

func TestStandardJSONBackendIsNamed(t *testing.T) {
	if StandardJSONBackend() == "" {
		t.Fatal("standard JSON backend name is empty")
	}
}
