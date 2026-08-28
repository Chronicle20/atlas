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

// TestCreateAssetCommandBody_ExplicitStats_RoundTrip pins that explicit
// per-stat values and an upgrade-slot count survive a marshal/unmarshal
// round-trip, so a craft's exact reagent-adjusted stats are not lost on the
// wire.
func TestCreateAssetCommandBody_ExplicitStats_RoundTrip(t *testing.T) {
	in := CreateAssetCommandBody{
		TemplateId:    1082002,
		Quantity:      1,
		Slots:         7,
		Strength:      3,
		WeaponAttack:  4,
		WeaponDefense: 6,
		HP:            15,
	}
	bs, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out CreateAssetCommandBody
	if err := json.Unmarshal(bs, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Slots != in.Slots {
		t.Errorf("Slots = %d, want %d", out.Slots, in.Slots)
	}
	if out.Strength != in.Strength {
		t.Errorf("Strength = %d, want %d", out.Strength, in.Strength)
	}
	if out.WeaponAttack != in.WeaponAttack {
		t.Errorf("WeaponAttack = %d, want %d", out.WeaponAttack, in.WeaponAttack)
	}
	if out.WeaponDefense != in.WeaponDefense {
		t.Errorf("WeaponDefense = %d, want %d", out.WeaponDefense, in.WeaponDefense)
	}
	if out.HP != in.HP {
		t.Errorf("HP = %d, want %d", out.HP, in.HP)
	}
}

// TestCreateAssetCommandBody_LegacyPayloadDecodesWithZeroStats pins that the
// stat/slot extension is additive: a pre-existing producer that never sends
// these fields decodes them as their zero value with no error, so every
// existing producer keeps working unchanged.
func TestCreateAssetCommandBody_LegacyPayloadDecodesWithZeroStats(t *testing.T) {
	legacy := `{"templateId":1082002,"quantity":1,"ownerId":0,"flag":0,"rechargeable":0}`
	var out CreateAssetCommandBody
	if err := json.Unmarshal([]byte(legacy), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Slots != 0 {
		t.Errorf("Slots = %d, want 0", out.Slots)
	}
	if out.Strength != 0 {
		t.Errorf("Strength = %d, want 0", out.Strength)
	}
	if out.WeaponAttack != 0 {
		t.Errorf("WeaponAttack = %d, want 0", out.WeaponAttack)
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
