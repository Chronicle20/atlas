package configuration

import (
	"testing"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestRpsRewardRoundTrip verifies that TransformRpsReward(ExtractRpsReward(m))
// round-trips the entryCostMeso/consolationMeso attributes and a 2-rung
// certificate ladder.
func TestRpsRewardRoundTrip(t *testing.T) {
	original := RpsRewardRestModel{
		Id:              "rps-rewards",
		EntryCostMeso:   1000,
		ConsolationMeso: 500,
		Ladder: []RpsRewardRungRestModel{
			{Rung: 1, ItemId: 4031332, Quantity: 1, Meso: 0},
			{Rung: 2, ItemId: 4031333, Quantity: 1, Meso: 0},
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

	if transformed.ConsolationMeso != original.ConsolationMeso {
		t.Errorf("ConsolationMeso mismatch: got %d, want %d", transformed.ConsolationMeso, original.ConsolationMeso)
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

func TestTransformRoute_PopulatesDerivedUuid(t *testing.T) {
	tid := uuid.MustParse("ec876921-c363-4cc6-9c51-5bb8d57f9553")
	data := map[string]interface{}{
		"id":         "boat-ellinia-orbis",
		"type":       "routes",
		"attributes": map[string]interface{}{"name": "boat-ellinia-orbis"},
	}
	rm, err := TransformRoute(tid, data)
	if err != nil {
		t.Fatalf("TransformRoute: %v", err)
	}
	want := tenant.DerivedId(tid, "routes", "boat-ellinia-orbis").String()
	if rm.Uuid != want {
		t.Fatalf("Uuid = %q, want %q", rm.Uuid, want)
	}
	if rm.Id != "boat-ellinia-orbis" {
		t.Fatalf("Id = %q, want the slug (uuid is additive)", rm.Id)
	}
}

func TestTransformInstanceRoute_PopulatesDerivedUuid(t *testing.T) {
	tid := uuid.MustParse("ec876921-c363-4cc6-9c51-5bb8d57f9553")
	data := map[string]interface{}{
		"id":         "temple-of-time-return-flight",
		"type":       "instance-routes",
		"attributes": map[string]interface{}{"name": "temple-of-time-return-flight"},
	}
	rm, err := TransformInstanceRoute(tid, data)
	if err != nil {
		t.Fatalf("TransformInstanceRoute: %v", err)
	}
	want := tenant.DerivedId(tid, "instance-routes", "temple-of-time-return-flight").String()
	if rm.Uuid != want {
		t.Fatalf("Uuid = %q, want %q", rm.Uuid, want)
	}
}

func TestTransformVessel_PopulatesDerivedUuid(t *testing.T) {
	tid := uuid.MustParse("ec876921-c363-4cc6-9c51-5bb8d57f9553")
	data := map[string]interface{}{
		"id":         "boat-ellinia-orbis",
		"type":       "vessels",
		"attributes": map[string]interface{}{"name": "boat-ellinia-orbis"},
	}
	rm, err := TransformVessel(tid, data)
	if err != nil {
		t.Fatalf("TransformVessel: %v", err)
	}
	want := tenant.DerivedId(tid, "vessels", "boat-ellinia-orbis").String()
	if rm.Uuid != want {
		t.Fatalf("Uuid = %q, want %q", rm.Uuid, want)
	}
}

// A route and an instance-route sharing one slug must not share a uuid.
func TestTransform_ResourcesDoNotCollideOnSharedSlug(t *testing.T) {
	tid := uuid.MustParse("ec876921-c363-4cc6-9c51-5bb8d57f9553")
	shared := map[string]interface{}{"id": "shared-slug", "attributes": map[string]interface{}{"name": "shared-slug"}}
	r, _ := TransformRoute(tid, shared)
	i, _ := TransformInstanceRoute(tid, shared)
	if r.Uuid == i.Uuid {
		t.Fatalf("route and instance-route collided on uuid %q", r.Uuid)
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
