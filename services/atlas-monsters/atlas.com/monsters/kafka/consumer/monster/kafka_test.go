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

func TestSetAggroCommandUnmarshal(t *testing.T) {
	raw := []byte(`{"worldId":1,"channelId":2,"mapId":100000000,"instance":"00000000-0000-0000-0000-000000000000","monsterId":4242,"type":"SET_AGGRO","body":{"characterId":777}}`)
	var c command[setAggroCommandBody]
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.Type != CommandTypeSetAggro {
		t.Errorf("Type = %s, want %s", c.Type, CommandTypeSetAggro)
	}
	if c.MonsterId != 4242 {
		t.Errorf("MonsterId = %d, want 4242", c.MonsterId)
	}
	if c.Body.CharacterId != 777 {
		t.Errorf("Body.CharacterId = %d, want 777", c.Body.CharacterId)
	}
}

func TestSetAggroCommandTypeConstant(t *testing.T) {
	if CommandTypeSetAggro != "SET_AGGRO" {
		t.Errorf("CommandTypeSetAggro = %s, want SET_AGGRO", CommandTypeSetAggro)
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

func TestSelfDestructCommandUnmarshal(t *testing.T) {
	raw := []byte(`{"worldId":0,"channelId":1,"mapId":100000000,"monsterId":7001,"type":"SELF_DESTRUCT","body":{"characterId":4242}}`)
	var c command[selfDestructCommandBody]
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if c.Type != CommandTypeSelfDestruct {
		t.Errorf("Type = %s, want %s", c.Type, CommandTypeSelfDestruct)
	}
	if c.MonsterId != 7001 {
		t.Errorf("MonsterId = %d, want 7001", c.MonsterId)
	}
	if c.Body.CharacterId != 4242 {
		t.Errorf("Body.CharacterId = %d, want 4242", c.Body.CharacterId)
	}
}

func TestSelfDestructCommandTypeValue(t *testing.T) {
	if CommandTypeSelfDestruct != "SELF_DESTRUCT" {
		t.Fatalf("CommandTypeSelfDestruct = %s, want SELF_DESTRUCT", CommandTypeSelfDestruct)
	}
}
