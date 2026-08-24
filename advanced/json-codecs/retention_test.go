package jsoncodecs

import "testing"

func TestRunRetentionExperiment(t *testing.T) {
	for _, codec := range Codecs() {
		t.Run(codec.Name(), func(t *testing.T) {
			result, err := RunRetentionExperiment(codec, 2, 128)
			if err != nil {
				t.Fatal(err)
			}
			if result.Documents != 2 || result.PaddingPerInput != 128 {
				t.Fatalf("unexpected result: %+v", result)
			}
			if result.LogicalKeepBytes != len("token-00000000")*2 {
				t.Fatalf("logical keep bytes = %d", result.LogicalKeepBytes)
			}
		})
	}
}

func TestRunRetentionExperimentRejectsInvalidInput(t *testing.T) {
	for _, tc := range []struct {
		name      string
		codec     Codec
		documents int
		padding   int
	}{
		{name: "nil codec", documents: 1},
		{name: "zero documents", codec: StandardLibrary},
		{name: "negative padding", codec: StandardLibrary, documents: 1, padding: -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := RunRetentionExperiment(tc.codec, tc.documents, tc.padding); err == nil {
				t.Fatal("invalid experiment succeeded")
			}
		})
	}
}
