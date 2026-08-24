//go:build goexperiment.jsonv2

package jsoncodecs

import (
	"bytes"
	"testing"
	"unicode/utf8"
)

func TestGoJSONV2IntentionalSemanticDifferences(t *testing.T) {
	t.Run("nil slice", func(t *testing.T) {
		value := struct {
			Items []string `json:"items"`
		}{}
		v1, err := StandardLibrary.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		v2, err := GoJSONV2.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(v1, []byte(`{"items":null}`)) || !bytes.Equal(v2, []byte(`{"items":[]}`)) {
			t.Fatalf("nil slice: v1=%s v2=%s", v1, v2)
		}
	})

	t.Run("duplicate name", func(t *testing.T) {
		input := []byte(`{"name":"first","name":"last"}`)
		var v1, v2 struct {
			Name string `json:"name"`
		}
		if err := StandardLibrary.Unmarshal(input, &v1); err != nil {
			t.Fatalf("v1 rejected duplicate name: %v", err)
		}
		if err := GoJSONV2.Unmarshal(input, &v2); err == nil {
			t.Fatal("v2 accepted duplicate name")
		}
	})

	t.Run("invalid UTF-8", func(t *testing.T) {
		input := []byte{'{', '"', 'n', 'a', 'm', 'e', '"', ':', '"', 0xff, '"', '}'}
		var v1, v2 struct {
			Name string `json:"name"`
		}
		if err := StandardLibrary.Unmarshal(input, &v1); err != nil {
			t.Fatalf("v1 rejected invalid UTF-8: %v", err)
		}
		if !utf8.ValidString(v1.Name) {
			t.Fatalf("v1 did not replace invalid UTF-8: %q", v1.Name)
		}
		if err := GoJSONV2.Unmarshal(input, &v2); err == nil {
			t.Fatal("v2 accepted invalid UTF-8")
		}
	})

	t.Run("field-name case", func(t *testing.T) {
		input := []byte(`{"NAME":"value"}`)
		var v1, v2 struct {
			Name string `json:"name"`
		}
		if err := StandardLibrary.Unmarshal(input, &v1); err != nil {
			t.Fatal(err)
		}
		if err := GoJSONV2.Unmarshal(input, &v2); err != nil {
			t.Fatal(err)
		}
		if v1.Name != "value" || v2.Name != "" {
			t.Fatalf("case matching: v1=%q v2=%q", v1.Name, v2.Name)
		}
	})
}
