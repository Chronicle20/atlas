package configuration

import (
	"testing"
)

// TestRpsRewardRoundTrip verifies that TransformRpsReward(ExtractRpsReward(m))
// round-trips the entryCostMeso attribute and a 2-rung ladder.
func TestRpsRewardRoundTrip(t *testing.T) {
	original := RpsRewardRestModel{
		Id:            "rps-rewards",
		EntryCostMeso: 1000,
		Ladder: []RpsRewardRungRestModel{
			{Rung: 1, ItemId: 0, Quantity: 0, Meso: 2000},
			{Rung: 2, ItemId: 0, Quantity: 0, Meso: 5000},
		},
	}

	extracted, err := ExtractRpsReward(original)
	if err != nil {
		t.Fatalf("ExtractRpsReward returned error: %v", err)
	}

	if extracted["type"] != "rps-rewards" {
		t.Fatalf("expected type %q, got %v", "rps-rewards", extracted["type"])
	}

	attributes, ok := extracted["attributes"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected attributes map, got %T", extracted["attributes"])
	}

	// Simulate the JSON:API wire format by wrapping the extracted map exactly
	// as it would be persisted/served (numeric attrs surface as float64 once
	// round-tripped through JSON, so mirror that here rather than relying on
	// the native Go int/uint32 types produced by Extract).
	wireData := map[string]interface{}{
		"id":         extracted["id"],
		"attributes": toFloat64Attributes(attributes),
	}

	transformed, err := TransformRpsReward(wireData)
	if err != nil {
		t.Fatalf("TransformRpsReward returned error: %v", err)
	}

	if transformed.EntryCostMeso != original.EntryCostMeso {
		t.Errorf("EntryCostMeso mismatch: got %d, want %d", transformed.EntryCostMeso, original.EntryCostMeso)
	}

	if len(transformed.Ladder) != len(original.Ladder) {
		t.Fatalf("Ladder length mismatch: got %d, want %d", len(transformed.Ladder), len(original.Ladder))
	}

	for i, rung := range original.Ladder {
		got := transformed.Ladder[i]
		if got.Rung != rung.Rung || got.ItemId != rung.ItemId || got.Quantity != rung.Quantity || got.Meso != rung.Meso {
			t.Errorf("Ladder[%d] mismatch: got %+v, want %+v", i, got, rung)
		}
	}
}

// toFloat64Attributes converts numeric attribute values (and nested ladder
// rung values) to float64, mirroring how they arrive after JSON
// marshal/unmarshal over the wire (encoding/json decodes all JSON numbers
// into map[string]interface{} as float64).
func toFloat64Attributes(attributes map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(attributes))
	for k, v := range attributes {
		switch val := v.(type) {
		case uint32:
			out[k] = float64(val)
		case []RpsRewardRungRestModel:
			rungs := make([]interface{}, 0, len(val))
			for _, r := range val {
				rungs = append(rungs, map[string]interface{}{
					"rung":     float64(r.Rung),
					"itemId":   float64(r.ItemId),
					"quantity": float64(r.Quantity),
					"meso":     float64(r.Meso),
				})
			}
			out[k] = rungs
		default:
			out[k] = v
		}
	}
	return out
}
