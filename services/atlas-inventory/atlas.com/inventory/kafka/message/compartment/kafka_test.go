package compartment

import (
	"encoding/json"
	"strings"
	"testing"
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
