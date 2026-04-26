package monster

import (
	"encoding/json"
	"testing"
)

func TestDamageCommandBody_DecodeNewShape(t *testing.T) {
	raw := []byte(`{"characterId":42,"damages":[100,200,300],"attackType":1}`)
	var body damageCommandBody
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if body.CharacterId != 42 {
		t.Fatalf("CharacterId = %d, want 42", body.CharacterId)
	}
	if len(body.Damages) != 3 || body.Damages[0] != 100 || body.Damages[1] != 200 || body.Damages[2] != 300 {
		t.Fatalf("Damages = %v, want [100 200 300]", body.Damages)
	}
	if body.AttackType != 1 {
		t.Fatalf("AttackType = %d, want 1", body.AttackType)
	}
}

func TestDamageCommandBody_MissingDamagesIsNil(t *testing.T) {
	raw := []byte(`{"characterId":42,"attackType":1}`)
	var body damageCommandBody
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if body.Damages != nil {
		t.Fatalf("Damages = %v, want nil for missing field", body.Damages)
	}
}

func TestDamageCommandBody_OldDamageFieldIgnored(t *testing.T) {
	// In-flight messages from the old shape have only "damage" (singular).
	// The new consumer must decode them with Damages == nil so the handler
	// no-ops them. Asserts the schema rename was a hard cut, not a coexist.
	raw := []byte(`{"characterId":42,"damage":500,"attackType":1}`)
	var body damageCommandBody
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if body.Damages != nil {
		t.Fatalf("Damages = %v, want nil when only legacy 'damage' field present", body.Damages)
	}
}
