package configuration

import (
	"encoding/json"
	"fmt"
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

func boolPtr(v bool) *bool { return &v }

func intPtr(v int) *int { return &v }

// wantIntPtr asserts an optional int attribute survived with the expected value.
func wantIntPtr(t *testing.T, name string, got *int, want int) {
	t.Helper()
	if got == nil {
		t.Errorf("%s: got nil, want %d", name, want)
		return
	}
	if *got != want {
		t.Errorf("%s: got %d, want %d", name, *got, want)
	}
}

// TestTradeConfigRoundTrip pins that ExtractTradeConfig -> TransformTradeConfig
// preserves every knob, including the taxTiers array — the one attribute that
// is a list of objects rather than a scalar, and therefore the one whose
// Transform arm cannot use the `attributes["x"].(float64)` shape.
func TestTradeConfigRoundTrip(t *testing.T) {
	original := TradeConfigRestModel{
		Id:         "trade-configs",
		TaxEnabled: boolPtr(true),
		TaxTiers: []TradeTaxTierRestModel{
			{Threshold: 100000000, Rate: 0.060},
			{Threshold: 100000, Rate: 0.008},
		},
		MaxStagedItems:            intPtr(9),
		MinTradeLevel:             intPtr(15),
		AttestationTimeoutSeconds: intPtr(5),
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
	wantIntPtr(t, "MaxStagedItems", got.MaxStagedItems, 9)
	wantIntPtr(t, "MinTradeLevel", got.MinTradeLevel, 15)
	wantIntPtr(t, "AttestationTimeoutSeconds", got.AttestationTimeoutSeconds, 5)

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
// empty payload yields nils rather than panicking, and — critically — that an
// absent attribute is nil rather than a zero value, so ExtractTradeConfig will
// omit it and the merge will keep whatever was stored.
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
	if got.MaxStagedItems != nil || got.MinTradeLevel != nil ||
		got.AttestationTimeoutSeconds != nil {
		t.Errorf("scalar knobs: got %+v, want all nil for absent attributes", got)
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
	wantIntPtr(t, "maxStagedItems", rm.MaxStagedItems, 9)
	wantIntPtr(t, "minTradeLevel", rm.MinTradeLevel, 0)
	wantIntPtr(t, "attestationTimeoutSeconds", rm.AttestationTimeoutSeconds, 5)

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

// storedTradeConfig is the JSONB shape a configured tenant carries: one JSON:API
// resource with the full attribute set. minTradeLevel is deliberately non-zero
// so that a wipe-to-zero is detectable — its default IS 0, so a zero here would
// hide exactly the corruption these tests exist to catch.
func storedTradeConfig() map[string]interface{} {
	return map[string]interface{}{
		"id":   "trade-configs",
		"type": "trade-configs",
		"attributes": map[string]interface{}{
			"taxEnabled": true,
			"taxTiers": []interface{}{
				map[string]interface{}{"threshold": float64(50000000), "rate": 0.11},
				map[string]interface{}{"threshold": float64(100000), "rate": 0.008},
			},
			"maxStagedItems":            float64(9),
			"minTradeLevel":             float64(15),
			"attestationTimeoutSeconds": float64(5),
		},
	}
}

// applyPatch runs one PATCH body through the real production path: the JSON:API
// decode the input handler performs, then ExtractTradeConfig, then the merge
// UpdateTradeConfig applies, then the JSONB round trip, then the Transform a
// later GET performs.
func applyPatch(t *testing.T, stored map[string]interface{}, body string) TradeConfigRestModel {
	t.Helper()

	var patch TradeConfigRestModel
	if err := jsonapi.Unmarshal([]byte(body), &patch); err != nil {
		t.Fatalf("jsonapi.Unmarshal: %v", err)
	}

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
	return got
}

// patchDocument builds a JSON:API PATCH body naming exactly one attribute.
func patchDocument(attribute string, value string) string {
	return fmt.Sprintf(`{"data":{"type":"trade-configs","id":"trade-configs","attributes":{%q:%s}}}`, attribute, value)
}

// TestPartialPatchThroughTheRealPathPreservesEveryOtherAttribute is the sweep
// the earlier round was missing. For EACH attribute in turn it PATCHes that one
// attribute through the real jsonapi.Unmarshal -> ExtractTradeConfig -> merge
// path and asserts every other stored attribute survives untouched.
//
// The earlier TestMergePreservesUnmentionedAttributesGenerally hand-built a
// partial `incoming` map and so only proved the merge function correct in
// isolation — it never exercised whether ExtractTradeConfig actually produces a
// partial map. It did not, for five of the six attributes, and the merge cannot
// rescue a key that arrives written at its zero value.
func TestPartialPatchThroughTheRealPathPreservesEveryOtherAttribute(t *testing.T) {
	cases := []struct {
		attribute string
		value     string
	}{
		{"taxEnabled", "false"},
		{"maxStagedItems", "3"},
		{"minTradeLevel", "40"},
		{"attestationTimeoutSeconds", "8"},
		{"taxTiers", `[{"threshold":7,"rate":0.5}]`},
	}

	for _, c := range cases {
		t.Run(c.attribute, func(t *testing.T) {
			got := applyPatch(t, storedTradeConfig(), patchDocument(c.attribute, c.value))

			if c.attribute != "taxEnabled" {
				if got.TaxEnabled == nil {
					t.Error("taxEnabled was wiped by a PATCH that never mentioned it")
				} else if !*got.TaxEnabled {
					t.Error("taxEnabled: got false, want the stored true to survive")
				}
			}
			if c.attribute != "maxStagedItems" {
				wantIntPtr(t, "maxStagedItems", got.MaxStagedItems, 9)
			}
			if c.attribute != "minTradeLevel" {
				wantIntPtr(t, "minTradeLevel", got.MinTradeLevel, 15)
			}
			if c.attribute != "attestationTimeoutSeconds" {
				wantIntPtr(t, "attestationTimeoutSeconds", got.AttestationTimeoutSeconds, 5)
			}
			if c.attribute != "taxTiers" {
				want := []TradeTaxTierRestModel{
					{Threshold: 50000000, Rate: 0.11},
					{Threshold: 100000, Rate: 0.008},
				}
				if len(got.TaxTiers) != len(want) {
					t.Fatalf("taxTiers: got %d tiers, want the stored %d to survive", len(got.TaxTiers), len(want))
				}
				for i := range want {
					if got.TaxTiers[i] != want[i] {
						t.Errorf("taxTiers[%d]: got %+v, want the stored %+v", i, got.TaxTiers[i], want[i])
					}
				}
			}
		})
	}
}

// TestPartialPatchAppliesThePatchedAttribute pins the other half of the sweep:
// the attribute the PATCH does name must actually change. Without this, an
// ExtractTradeConfig that omitted everything would pass the preservation sweep.
func TestPartialPatchAppliesThePatchedAttribute(t *testing.T) {
	stored := storedTradeConfig()

	if got := applyPatch(t, stored, patchDocument("maxStagedItems", "3")); got.MaxStagedItems == nil || *got.MaxStagedItems != 3 {
		t.Errorf("maxStagedItems: got %v, want the patched 3", got.MaxStagedItems)
	}
	if got := applyPatch(t, stored, patchDocument("minTradeLevel", "40")); got.MinTradeLevel == nil || *got.MinTradeLevel != 40 {
		t.Errorf("minTradeLevel: got %v, want the patched 40", got.MinTradeLevel)
	}
	if got := applyPatch(t, stored, patchDocument("attestationTimeoutSeconds", "8")); got.AttestationTimeoutSeconds == nil || *got.AttestationTimeoutSeconds != 8 {
		t.Errorf("attestationTimeoutSeconds: got %v, want the patched 8", got.AttestationTimeoutSeconds)
	}

	got := applyPatch(t, stored, patchDocument("taxTiers", `[{"threshold":7,"rate":0.5}]`))
	if len(got.TaxTiers) != 1 {
		t.Fatalf("taxTiers: got %d tiers, want the patched 1", len(got.TaxTiers))
	}
	if got.TaxTiers[0] != (TradeTaxTierRestModel{Threshold: 7, Rate: 0.5}) {
		t.Errorf("taxTiers[0]: got %+v, want {7 0.5}", got.TaxTiers[0])
	}
}

// TestExplicitZeroIsTransmittedAndStored pins that the optionality fix does not
// trade one silent-corruption bug for another: an operator who deliberately
// clears the level gate with `minTradeLevel: 0` must have that stored, not
// discarded as "absent". api2go marshals via encoding/json, whose omitempty
// omits only a nil pointer — never a pointer to 0.
func TestExplicitZeroIsTransmittedAndStored(t *testing.T) {
	// The wire must carry the explicit zero out.
	raw, err := jsonapi.Marshal(TradeConfigRestModel{Id: "trade-configs", MinTradeLevel: intPtr(0)})
	if err != nil {
		t.Fatalf("jsonapi.Marshal: %v", err)
	}
	want := `{"data":{"type":"trade-configs","id":"trade-configs","attributes":{"minTradeLevel":0}}}`
	if string(raw) != want {
		t.Errorf("an explicit zero did not survive marshalling:\n got: %s\nwant: %s", raw, want)
	}

	// And the inbound path must store it over a non-zero stored value.
	got := applyPatch(t, storedTradeConfig(), patchDocument("minTradeLevel", "0"))
	if got.MinTradeLevel == nil {
		t.Fatal("minTradeLevel: got nil, want the explicit 0 to be stored")
	}
	if *got.MinTradeLevel != 0 {
		t.Errorf("minTradeLevel: got %d, want the explicit 0 to overwrite the stored 15", *got.MinTradeLevel)
	}
}

// TestExplicitFalseIsTransmittedAndStored is the taxEnabled counterpart: the
// meso-tax off switch must stay turnable.
func TestExplicitFalseIsTransmittedAndStored(t *testing.T) {
	got := applyPatch(t, storedTradeConfig(), patchDocument("taxEnabled", "false"))
	if got.TaxEnabled == nil {
		t.Fatal("taxEnabled: got nil, want an explicit false")
	}
	if *got.TaxEnabled {
		t.Error("taxEnabled: got true, want the explicit false to overwrite the stored true")
	}
}

// TestMergePreservesUnmentionedAttributes pins mergeTradeConfigAttributes in
// isolation: an omitted attribute survives, a named one wins (including an
// explicit zero), and the stored map is not mutated in place. This covers the
// merge function only — that production actually hands it a partial map is what
// TestPartialPatchThroughTheRealPathPreservesEveryOtherAttribute covers.
func TestMergePreservesUnmentionedAttributes(t *testing.T) {
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

	if attributes["attestationTimeoutSeconds"] != float64(5) {
		t.Errorf("attestationTimeoutSeconds: got %v, want the stored 5 to survive", attributes["attestationTimeoutSeconds"])
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
const tradeConfigWireDocument = `{"data":{"type":"trade-configs","id":"trade-configs","attributes":{"taxEnabled":true,"taxTiers":[{"threshold":100000000,"rate":0.06},{"threshold":100000,"rate":0.008}],"maxStagedItems":9,"minTradeLevel":0,"attestationTimeoutSeconds":5}}}`

// TestTradeConfigWireShape pins the serialized JSON:API form, in particular that
// api2go nests taxTiers as an array of {threshold, rate} objects directly under
// attributes. A structural change here silently breaks atlas-trades' decode.
// minTradeLevel is 0 here on purpose: it must still appear on the wire, since
// the knob is a *int and omitempty omits only nil.
func TestTradeConfigWireShape(t *testing.T) {
	rm := TradeConfigRestModel{
		Id:         "trade-configs",
		TaxEnabled: boolPtr(true),
		TaxTiers: []TradeTaxTierRestModel{
			{Threshold: 100000000, Rate: 0.060},
			{Threshold: 100000, Rate: 0.008},
		},
		MaxStagedItems:            intPtr(9),
		MinTradeLevel:             intPtr(0),
		AttestationTimeoutSeconds: intPtr(5),
	}

	raw, err := jsonapi.Marshal(rm)
	if err != nil {
		t.Fatalf("jsonapi.Marshal: %v", err)
	}

	if string(raw) != tradeConfigWireDocument {
		t.Errorf("wire document mismatch:\n got: %s\nwant: %s", raw, tradeConfigWireDocument)
	}
}
