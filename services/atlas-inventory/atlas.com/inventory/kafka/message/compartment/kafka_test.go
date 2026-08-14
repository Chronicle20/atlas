package compartment

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestCreateAssetCommandBody_UseAverageStats_RoundTrip(t *testing.T) {
	in := CreateAssetCommandBody{TemplateId: 1, Quantity: 1, UseAverageStats: true}
	bs, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(bs), `"useAverageStats":true`) {
		t.Fatalf("expected useAverageStats:true in JSON, got %s", string(bs))
	}
	var out CreateAssetCommandBody
	if err := json.Unmarshal(bs, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !out.UseAverageStats {
		t.Fatalf("expected UseAverageStats=true after round-trip, got false")
	}
}

func TestCreateAssetCommandBody_UseAverageStats_OmitEmpty(t *testing.T) {
	in := CreateAssetCommandBody{TemplateId: 1, Quantity: 1, UseAverageStats: false}
	bs, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(bs), `"useAverageStats"`) {
		t.Fatalf("expected useAverageStats to be omitted when false, got %s", string(bs))
	}
}

// TestExtendExpirationCommandBody_JsonTags pins the wire shape of
// ExtendExpirationCommandBody. This body is hand-duplicated in
// atlas-saga-orchestrator's copy of this package; a field rename or json tag
// drift on either side compiles cleanly but decodes into a zero-valued body
// at runtime, so this test exists to catch that drift on THIS side.
func TestExtendExpirationCommandBody_JsonTags(t *testing.T) {
	exp := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	in := ExtendExpirationCommandBody{
		Slot:               5,
		Expiration:         exp,
		ExtenderTemplateId: 5500001,
	}
	bs, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(bs, &raw); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}
	for _, key := range []string{"slot", "expiration", "extenderTemplateId"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("expected json key %q in %s", key, string(bs))
		}
	}

	var out ExtendExpirationCommandBody
	if err := json.Unmarshal(bs, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Slot != in.Slot {
		t.Errorf("Slot = %d, want %d", out.Slot, in.Slot)
	}
	if !out.Expiration.Equal(in.Expiration) {
		t.Errorf("Expiration = %v, want %v", out.Expiration, in.Expiration)
	}
	if out.ExtenderTemplateId != in.ExtenderTemplateId {
		t.Errorf("ExtenderTemplateId = %d, want %d", out.ExtenderTemplateId, in.ExtenderTemplateId)
	}
}
