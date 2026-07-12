package character

import (
	"encoding/json"
	"testing"
)

func TestTemporalDataJSONRoundTripIncludesFh(t *testing.T) {
	in := temporalData{x: -12, y: 250, fh: 37, stance: 4}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out temporalData
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.X() != -12 || out.Y() != 250 || out.Fh() != 37 || out.Stance() != 4 {
		t.Errorf("round trip mismatch: got x=%d y=%d fh=%d stance=%d", out.X(), out.Y(), out.Fh(), out.Stance())
	}
}

func TestTemporalDataUnmarshalWithoutFhDefaultsZero(t *testing.T) {
	// Entries written before this change have no "fh" key — must decode as 0.
	var out temporalData
	if err := json.Unmarshal([]byte(`{"x":5,"y":6,"stance":2}`), &out); err != nil {
		t.Fatalf("unmarshal legacy payload: %v", err)
	}
	if out.Fh() != 0 || out.X() != 5 {
		t.Errorf("legacy decode mismatch: fh=%d x=%d", out.Fh(), out.X())
	}
}
