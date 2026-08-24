package lab

import "testing"

func FuzzBuildMembershipNoFalseNegatives(f *testing.F) {
	f.Add([]byte("alpha"))
	f.Add([]byte{})
	f.Add([]byte{0x00, 0xff, 0x80})

	f.Fuzz(func(t *testing.T, key []byte) {
		filter, err := BuildMembership(
			[][]byte{key},
			Config{PlannedItems: 1, TargetFalsePositiveRate: 0.01},
		)
		if err != nil {
			t.Fatalf("BuildMembership(%q): %v", key, err)
		}
		if !filter.MayContain(key) {
			t.Errorf("MayContain(%q) = false, want true", key)
		}
	})
}

func FuzzFilteredAndExactSegmentsAgree(f *testing.F) {
	f.Add("alpha")
	f.Add("missing")
	f.Add(string([]byte{0x00, 0xff, 0x80}))

	f.Fuzz(func(t *testing.T, key string) {
		records := []Record{
			{Key: "alpha", Value: "A"},
			{Key: "binary-\x00", Value: "B"},
			{Key: "omega", Value: "Z"},
		}
		cfg := Config{PlannedItems: uint64(len(records)), TargetFalsePositiveRate: 0.01}
		filtered, err := NewSegment(records, &cfg)
		if err != nil {
			t.Fatalf("NewSegment(filtered): %v", err)
		}
		exact, err := NewSegment(records, nil)
		if err != nil {
			t.Fatalf("NewSegment(exact): %v", err)
		}

		gotRecord, gotFound, _ := LookupSegments([]*Segment{filtered}, key)
		wantRecord, wantFound, _ := LookupSegments([]*Segment{exact}, key)
		if gotFound != wantFound || gotRecord != wantRecord {
			t.Errorf(
				"LookupSegments(%q) = (%#v, %t), want (%#v, %t)",
				key,
				gotRecord,
				gotFound,
				wantRecord,
				wantFound,
			)
		}
	})
}
