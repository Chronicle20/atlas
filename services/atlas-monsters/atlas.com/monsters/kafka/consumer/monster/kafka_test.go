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

func TestUseBasicAttackCommandBody_Decode(t *testing.T) {
	raw := []byte(`{"attackPos":1}`)
	var body useBasicAttackCommandBody
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.AttackPos != 1 {
		t.Fatalf("AttackPos = %d, want 1", body.AttackPos)
	}
}

func TestSpawnFieldCommandBody_Decode(t *testing.T) {
	raw := []byte(`{"monsterId":100100,"x":250,"y":-130,"fh":7,"team":0}`)
	var body spawnFieldCommandBody
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.MonsterId != 100100 {
		t.Errorf("MonsterId = %d, want 100100", body.MonsterId)
	}
	if body.X != 250 || body.Y != -130 {
		t.Errorf("position = (%d, %d), want (250, -130)", body.X, body.Y)
	}
	if body.Fh != 7 {
		t.Errorf("Fh = %d, want 7", body.Fh)
	}
	if body.Team != 0 {
		t.Errorf("Team = %d, want 0", body.Team)
	}
}
