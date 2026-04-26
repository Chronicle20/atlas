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
