package configuration

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/jtumidanski/api2go/jsonapi"
)

// jsonRoundTrip marshals and re-parses a config map so its values arrive as the
// interface{} shapes Transform sees when reading a JSONB column back out —
// float64 numbers and []interface{} arrays, never the concrete Go types Extract
// produced.
func jsonRoundTrip(in map[string]interface{}) (map[string]interface{}, error) {
	raw, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// TestTradeConfigRoundTrip pins that ExtractTradeConfig -> TransformTradeConfig
// preserves every knob, including the taxTiers array — the one attribute that
// is a list of objects rather than a scalar, and therefore the one whose
// Transform arm cannot use the `attributes["x"].(float64)` shape.
func TestTradeConfigRoundTrip(t *testing.T) {
	enabled := true
	original := TradeConfigRestModel{
		Id:         "trade-configs",
		TaxEnabled: &enabled,
		TaxTiers: []TradeTaxTierRestModel{
			{Threshold: 100000000, Rate: 0.060},
			{Threshold: 100000, Rate: 0.008},
		},
		MaxStagedItems:            9,
		MinTradeLevel:             15,
		ReservationTtlSeconds:     300,
		AttestationTimeoutSeconds: 5,
	}

	extracted, err := ExtractTradeConfig(original)
	if err != nil {
		t.Fatalf("ExtractTradeConfig: %v", err)
	}

	// Round-trip through JSON so the attributes arrive as the interface{}
	// shapes Transform actually sees at runtime (float64 numbers, []interface{}
	// arrays) rather than the concrete Go types Extract produced.
	round, err := jsonRoundTrip(extracted)
	if err != nil {
		t.Fatalf("json round trip: %v", err)
	}

	got, err := TransformTradeConfig(round)
	if err != nil {
		t.Fatalf("TransformTradeConfig: %v", err)
	}

	if got.Id != original.Id {
		t.Errorf("Id: got %q, want %q", got.Id, original.Id)
	}
	if got.TaxEnabled == nil {
		t.Error("TaxEnabled: got nil, want a pointer to true")
	} else if *got.TaxEnabled != *original.TaxEnabled {
		t.Errorf("TaxEnabled: got %v, want %v", *got.TaxEnabled, *original.TaxEnabled)
	}
	if got.MaxStagedItems != original.MaxStagedItems {
		t.Errorf("MaxStagedItems: got %d, want %d", got.MaxStagedItems, original.MaxStagedItems)
	}
	if got.MinTradeLevel != original.MinTradeLevel {
		t.Errorf("MinTradeLevel: got %d, want %d", got.MinTradeLevel, original.MinTradeLevel)
	}
	if got.ReservationTtlSeconds != original.ReservationTtlSeconds {
		t.Errorf("ReservationTtlSeconds: got %d, want %d", got.ReservationTtlSeconds, original.ReservationTtlSeconds)
	}
	if got.AttestationTimeoutSeconds != original.AttestationTimeoutSeconds {
		t.Errorf("AttestationTimeoutSeconds: got %d, want %d", got.AttestationTimeoutSeconds, original.AttestationTimeoutSeconds)
	}
	if len(got.TaxTiers) != len(original.TaxTiers) {
		t.Fatalf("TaxTiers: got %d tiers, want %d", len(got.TaxTiers), len(original.TaxTiers))
	}
	for i := range original.TaxTiers {
		if got.TaxTiers[i] != original.TaxTiers[i] {
			t.Errorf("TaxTiers[%d]: got %+v, want %+v", i, got.TaxTiers[i], original.TaxTiers[i])
		}
	}
}

// TestTradeConfigTransformToleratesMissingAttributes pins that a malformed or
// empty payload yields zeros rather than panicking — atlas-trades folds zeros
// back to its shipped defaults.
func TestTradeConfigTransformToleratesMissingAttributes(t *testing.T) {
	got, err := TransformTradeConfig(map[string]interface{}{"id": "trade-configs"})
	if err != nil {
		t.Fatalf("TransformTradeConfig: %v", err)
	}
	if got.TaxEnabled != nil {
		t.Errorf("TaxEnabled: got %v, want nil for an absent attribute", *got.TaxEnabled)
	}
	if len(got.TaxTiers) != 0 {
		t.Errorf("TaxTiers: got %d tiers, want 0", len(got.TaxTiers))
	}
	if got.MaxStagedItems != 0 || got.ReservationTtlSeconds != 0 || got.AttestationTimeoutSeconds != 0 {
		t.Errorf("scalar knobs: got %+v, want zeros", got)
	}
}

// TestSeedTradeConfigFileMatchesDesignDefaults pins that the shipped seed file
// parses through the real loader and carries exactly the design §8 values. It
// also pins the JSON:API envelope: SeedTradeConfigs feeds each loaded file
// straight to CreateTradeConfigAndEmit, which requires the top-level
// id/type/attributes shape — a flat attributes-only file would silently seed a
// config that Transform reads as all-zero.
func TestSeedTradeConfigFileMatchesDesignDefaults(t *testing.T) {
	t.Setenv("TRADE_CONFIGS_SEED_PATH", filepath.Join("..", "..", "..", "configurations", "trade-configs"))

	configs, errs := LoadTradeConfigFiles()
	if len(errs) != 0 {
		t.Fatalf("LoadTradeConfigFiles: %v", errs)
	}
	if len(configs) != 1 {
		t.Fatalf("expected exactly one seed file, got %d", len(configs))
	}

	if id, _ := configs[0]["id"].(string); id == "" {
		t.Error("seed file has no top-level id — CreateTradeConfig would store an unaddressable config")
	}
	if typ, _ := configs[0]["type"].(string); typ != "trade-configs" {
		t.Errorf("seed file type: got %q, want \"trade-configs\"", typ)
	}

	rm, err := TransformTradeConfig(configs[0])
	if err != nil {
		t.Fatalf("TransformTradeConfig: %v", err)
	}

	if rm.TaxEnabled == nil {
		t.Error("taxEnabled: got nil, want true")
	} else if !*rm.TaxEnabled {
		t.Error("taxEnabled: got false, want true")
	}
	if rm.MaxStagedItems != 9 {
		t.Errorf("maxStagedItems: got %d, want 9", rm.MaxStagedItems)
	}
	if rm.MinTradeLevel != 0 {
		t.Errorf("minTradeLevel: got %d, want 0", rm.MinTradeLevel)
	}
	if rm.ReservationTtlSeconds != 300 {
		t.Errorf("reservationTtlSeconds: got %d, want 300", rm.ReservationTtlSeconds)
	}
	if rm.AttestationTimeoutSeconds != 5 {
		t.Errorf("attestationTimeoutSeconds: got %d, want 5", rm.AttestationTimeoutSeconds)
	}

	want := []TradeTaxTierRestModel{
		{Threshold: 100000000, Rate: 0.060},
		{Threshold: 25000000, Rate: 0.050},
		{Threshold: 10000000, Rate: 0.040},
		{Threshold: 5000000, Rate: 0.030},
		{Threshold: 1000000, Rate: 0.018},
		{Threshold: 100000, Rate: 0.008},
	}
	if len(rm.TaxTiers) != len(want) {
		t.Fatalf("taxTiers: got %d tiers, want %d", len(rm.TaxTiers), len(want))
	}
	for i := range want {
		if rm.TaxTiers[i] != want[i] {
			t.Errorf("taxTiers[%d]: got %+v, want %+v", i, rm.TaxTiers[i], want[i])
		}
	}
}

// storedTradeConfig is the JSONB shape a seeded tenant carries: one JSON:API
// resource with the full design §8 attribute set.
func storedTradeConfig() map[string]interface{} {
	return map[string]interface{}{
		"id":   "trade-configs",
		"type": "trade-configs",
		"attributes": map[string]interface{}{
			"taxEnabled":                true,
			"taxTiers":                  []interface{}{map[string]interface{}{"threshold": float64(100000), "rate": 0.008}},
			"maxStagedItems":            float64(9),
			"minTradeLevel":             float64(0),
			"reservationTtlSeconds":     float64(300),
			"attestationTimeoutSeconds": float64(5),
		},
	}
}

// TestPartialPatchPreservesTaxEnabled pins the fix for the PATCH hazard: an
// operator who PATCHes only maxStagedItems must not silently disable the meso
// tax tenant-wide. The request decodes into the full TradeConfigRestModel with
// TaxEnabled nil (the attribute was never sent); ExtractTradeConfig omits the
// key, and the merge carries the stored true forward.
func TestPartialPatchPreservesTaxEnabled(t *testing.T) {
	stored := storedTradeConfig()

	// What jsonapi.Unmarshal produces for a body that names only maxStagedItems.
	patch := TradeConfigRestModel{Id: "trade-configs", MaxStagedItems: 3}

	incoming, err := ExtractTradeConfig(patch)
	if err != nil {
		t.Fatalf("ExtractTradeConfig: %v", err)
	}
	if _, present := incoming["attributes"].(map[string]interface{})["taxEnabled"]; present {
		t.Fatal("ExtractTradeConfig emitted taxEnabled for a nil pointer")
	}

	// UpdateTradeConfig marshals the merged config into the JSONB column and a
	// later GET unmarshals it back, so round-trip here too — otherwise the
	// numbers stay Go ints and Transform's float64 assertions all miss.
	merged, err := jsonRoundTrip(mergeTradeConfigAttributes(stored, incoming))
	if err != nil {
		t.Fatalf("json round trip: %v", err)
	}

	got, err := TransformTradeConfig(merged)
	if err != nil {
		t.Fatalf("TransformTradeConfig: %v", err)
	}
	if got.TaxEnabled == nil {
		t.Fatal("taxEnabled was dropped by a PATCH that never mentioned it")
	}
	if !*got.TaxEnabled {
		t.Error("taxEnabled: got false, want the stored true to survive the partial PATCH")
	}
	if got.MaxStagedItems != 3 {
		t.Errorf("maxStagedItems: got %d, want the patched 3", got.MaxStagedItems)
	}
}

// TestPatchCanStillDisableTheTax pins the other half: an explicit
// taxEnabled=false is a non-nil pointer, is serialised, and overwrites the
// stored true. Without this the fix above would make the knob unturnable.
func TestPatchCanStillDisableTheTax(t *testing.T) {
	stored := storedTradeConfig()

	disabled := false
	patch := TradeConfigRestModel{Id: "trade-configs", TaxEnabled: &disabled, MaxStagedItems: 9}

	incoming, err := ExtractTradeConfig(patch)
	if err != nil {
		t.Fatalf("ExtractTradeConfig: %v", err)
	}

	merged, err := jsonRoundTrip(mergeTradeConfigAttributes(stored, incoming))
	if err != nil {
		t.Fatalf("json round trip: %v", err)
	}

	got, err := TransformTradeConfig(merged)
	if err != nil {
		t.Fatalf("TransformTradeConfig: %v", err)
	}
	if got.TaxEnabled == nil {
		t.Fatal("taxEnabled: got nil, want an explicit false")
	}
	if *got.TaxEnabled {
		t.Error("taxEnabled: got true, want the explicit false to win")
	}
}

// TestMergePreservesUnmentionedAttributesGenerally pins that the merge is not
// taxEnabled-specific: any attribute the incoming config omits survives, and
// any attribute it names wins — including an explicit zero.
func TestMergePreservesUnmentionedAttributesGenerally(t *testing.T) {
	stored := storedTradeConfig()
	incoming := map[string]interface{}{
		"id":   "trade-configs",
		"type": "trade-configs",
		"attributes": map[string]interface{}{
			"minTradeLevel": float64(0),
			"invented":      "kept",
		},
	}

	merged := mergeTradeConfigAttributes(stored, incoming)
	attributes, ok := merged["attributes"].(map[string]interface{})
	if !ok {
		t.Fatal("merged config has no attributes map")
	}

	if attributes["reservationTtlSeconds"] != float64(300) {
		t.Errorf("reservationTtlSeconds: got %v, want the stored 300 to survive", attributes["reservationTtlSeconds"])
	}
	if attributes["minTradeLevel"] != float64(0) {
		t.Errorf("minTradeLevel: got %v, want the incoming explicit 0 to win", attributes["minTradeLevel"])
	}
	if attributes["invented"] != "kept" {
		t.Errorf("invented: got %v, want the incoming value", attributes["invented"])
	}

	// The stored map must not be mutated in place.
	storedAttributes := stored["attributes"].(map[string]interface{})
	if _, present := storedAttributes["invented"]; present {
		t.Error("mergeTradeConfigAttributes mutated the stored config in place")
	}
}

// tradeConfigWireDocument is the exact JSON:API document atlas-tenants emits for
// GET /tenants/{id}/configurations/trade-configs and that atlas-trades decodes.
// The identical literal is pinned on the consuming side in
// services/atlas-trades/atlas.com/trades/configuration/rest_test.go — the two
// tests together are the only thing that keeps the nested taxTiers object array
// symmetric across the module boundary.
const tradeConfigWireDocument = `{"data":{"type":"trade-configs","id":"trade-configs","attributes":{"taxEnabled":true,"taxTiers":[{"threshold":100000000,"rate":0.06},{"threshold":100000,"rate":0.008}],"maxStagedItems":9,"minTradeLevel":0,"reservationTtlSeconds":300,"attestationTimeoutSeconds":5}}}`

// TestTradeConfigWireShape pins the serialized JSON:API form, in particular that
// api2go nests taxTiers as an array of {threshold, rate} objects directly under
// attributes. A structural change here silently breaks atlas-trades' decode.
func TestTradeConfigWireShape(t *testing.T) {
	enabled := true
	rm := TradeConfigRestModel{
		Id:         "trade-configs",
		TaxEnabled: &enabled,
		TaxTiers: []TradeTaxTierRestModel{
			{Threshold: 100000000, Rate: 0.060},
			{Threshold: 100000, Rate: 0.008},
		},
		MaxStagedItems:            9,
		MinTradeLevel:             0,
		ReservationTtlSeconds:     300,
		AttestationTimeoutSeconds: 5,
	}

	raw, err := jsonapi.Marshal(rm)
	if err != nil {
		t.Fatalf("jsonapi.Marshal: %v", err)
	}

	if string(raw) != tradeConfigWireDocument {
		t.Errorf("wire document mismatch:\n got: %s\nwant: %s", raw, tradeConfigWireDocument)
	}
}
