package configuration

import (
	"encoding/json"
	"path/filepath"
	"testing"
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
	original := TradeConfigRestModel{
		Id:         "trade-configs",
		TaxEnabled: true,
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
	if got.TaxEnabled != original.TaxEnabled {
		t.Errorf("TaxEnabled: got %v, want %v", got.TaxEnabled, original.TaxEnabled)
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
	if got.TaxEnabled {
		t.Error("TaxEnabled: got true, want false for an absent attribute")
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

	if !rm.TaxEnabled {
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
