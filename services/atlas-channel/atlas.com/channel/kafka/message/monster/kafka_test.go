package monster

import (
	"encoding/json"
	"testing"
)

func TestDamageCommandBody_EncodeNewShape(t *testing.T) {
	body := DamageCommandBody{
		CharacterId: 42,
		Damages:     []uint32{100, 200, 300},
		AttackType:  1,
	}
	out, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	want := `{"characterId":42,"damages":[100,200,300],"attackType":1}`
	if string(out) != want {
		t.Fatalf("got %s, want %s", out, want)
	}
}

func TestDamageCommandBody_DecodeRoundTrip(t *testing.T) {
	in := DamageCommandBody{
		CharacterId: 7,
		Damages:     []uint32{1, 2},
		AttackType:  0,
	}
	raw, _ := json.Marshal(in)
	var got DamageCommandBody
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got.CharacterId != 7 || len(got.Damages) != 2 || got.Damages[0] != 1 || got.Damages[1] != 2 || got.AttackType != 0 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}
}

func TestStatusEventStartControlBody_DecodesControllerHasAggro(t *testing.T) {
	raw := []byte(`{"actorId":42,"x":1,"y":2,"stance":3,"fh":4,"team":5,"controllerHasAggro":true}`)
	var body StatusEventStartControlBody
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !body.ControllerHasAggro {
		t.Errorf("expected ControllerHasAggro=true, got %v", body.ControllerHasAggro)
	}
	if body.ActorId != 42 {
		t.Errorf("expected ActorId=42, got %d", body.ActorId)
	}
}

func TestStatusEventStartControlBody_LegacyDefaultsFalse(t *testing.T) {
	raw := []byte(`{"actorId":42,"x":1,"y":2,"stance":3,"fh":4,"team":5}`)
	var body StatusEventStartControlBody
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.ControllerHasAggro {
		t.Errorf("missing field should default to false")
	}
}

func TestStatusEventAggroChangedBody_RoundTrip(t *testing.T) {
	body := StatusEventAggroChangedBody{ControllerCharacterId: 7, ControllerHasAggro: true}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"controllerCharacterId":7,"controllerHasAggro":true}`
	if string(raw) != want {
		t.Errorf("got %s, want %s", string(raw), want)
	}
	var got StatusEventAggroChangedBody
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.ControllerCharacterId != 7 || !got.ControllerHasAggro {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

func TestEventStatusAggroChangedConstant(t *testing.T) {
	if EventStatusAggroChanged != "AGGRO_CHANGED" {
		t.Errorf("EventStatusAggroChanged=%q, want AGGRO_CHANGED", EventStatusAggroChanged)
	}
}
