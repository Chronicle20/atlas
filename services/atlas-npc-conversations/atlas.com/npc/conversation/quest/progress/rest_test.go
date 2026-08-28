package progress

import "testing"

// TestProgressRestRoundTrip verifies that RestModel/Model/Extract round-trip
// a quest progress entry without loss, and in particular that Progress stays
// a string and is never coerced to a numeric type.
func TestProgressRestRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		rm   RestModel
	}{
		{"numeric progress", RestModel{Id: 1, InfoNumber: 0, Progress: "72"}},
		{"text progress", RestModel{Id: 2, InfoNumber: 0, Progress: "Open Sesame"}},
		{"named info number", RestModel{Id: 3, InfoNumber: 9300285, Progress: "0"}},
		{"empty progress", RestModel{Id: 4, InfoNumber: 0, Progress: ""}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, err := Extract(tc.rm)
			if err != nil {
				t.Fatalf("Extract: %v", err)
			}
			if m.InfoNumber() != tc.rm.InfoNumber {
				t.Errorf("InfoNumber mismatch: got %d want %d", m.InfoNumber(), tc.rm.InfoNumber)
			}
			if m.Progress() != tc.rm.Progress {
				t.Errorf("Progress mismatch: got %q want %q", m.Progress(), tc.rm.Progress)
			}

			var rt RestModel
			if err := rt.SetID(tc.rm.GetID()); err != nil {
				t.Fatalf("SetID: %v", err)
			}
			if rt.Id != tc.rm.Id {
				t.Errorf("Id round trip mismatch: got %d want %d", rt.Id, tc.rm.Id)
			}
		})
	}
}

// TestGetName asserts the JSON:API type name matches the server's own
// GetName() (services/atlas-quest/atlas.com/quest/quest/progress/rest.go);
// a mismatch makes every decode return an empty collection with no error.
func TestGetName(t *testing.T) {
	got := RestModel{}.GetName()
	want := "progress"
	if got != want {
		t.Errorf("GetName mismatch: got %q want %q", got, want)
	}
}
