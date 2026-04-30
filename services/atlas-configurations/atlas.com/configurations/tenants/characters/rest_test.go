package characters

import (
	"encoding/json"
	"testing"
)

func TestRestModel_BackwardsCompat_NoPresets(t *testing.T) {
	in := []byte(`{"templates":[]}`)
	var out RestModel
	if err := json.Unmarshal(in, &out); err != nil {
		t.Fatalf("unmarshal legacy: %v", err)
	}
	// Acceptable for the field to be nil; the orchestrator coerces nil → []
	// at read-time. Just confirm no panic and no leftover noise.
	if out.Presets != nil && len(out.Presets) != 0 {
		t.Fatalf("legacy doc should not yield non-empty presets, got %+v", out.Presets)
	}
}

func TestRestModel_PresetsRoundTrip(t *testing.T) {
	in := []byte(`{"templates":[],"presets":[{"id":"abc","attributes":{"name":"x","jobId":112}}]}`)
	var out RestModel
	if err := json.Unmarshal(in, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(out.Presets) != 1 || out.Presets[0].Id != "abc" || out.Presets[0].Attributes.JobId != 112 {
		t.Fatalf("preset did not decode: %+v", out)
	}
}
