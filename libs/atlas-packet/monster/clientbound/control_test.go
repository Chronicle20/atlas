package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=monster/clientbound/MonsterControl version=gms_v83 ida=0x679777
// packet-audit:verify packet=monster/clientbound/MonsterControl version=gms_v87 ida=0x6b52c3
// packet-audit:verify packet=monster/clientbound/MonsterControl version=gms_v95 ida=0x658d10
// packet-audit:verify packet=monster/clientbound/MonsterControl version=jms_v185 ida=0x6f8b84
// packet-audit:verify packet=monster/clientbound/MonsterControl version=gms_v84 ida=0x69030d
func TestMonsterControlActiveInit(t *testing.T) {
	m := model.NewMonster(100, 200, 5, 300, model.MonsterAppearTypeRegen, 0)
	// ActiveInit is the "passive controller" assignment — no aggro responsibility.
	input := NewMonsterControl(ControlTypeActiveInit, 5001, 100100, m, false)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestMonsterControlReset(t *testing.T) {
	m := model.NewMonster(100, 200, 5, 300, model.MonsterAppearTypeRegen, 0)
	// Reset doesn't reach the aggro/monsterId tail at all; aggro arg unused.
	input := NewMonsterControl(ControlTypeReset, 5001, 100100, m, false)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestMonsterControlActiveRequest(t *testing.T) {
	m := model.NewMonster(100, 200, 5, 300, model.MonsterAppearTypeRegen, 0)
	// ActiveRequest is the "aggro controller" assignment — controller reports
	// damage events.
	input := NewMonsterControl(ControlTypeActiveRequest, 5001, 100100, m, true)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestMonsterControlBytesV79 pins the exact wire bytes against the v79 client
// read order in CMobPool::OnMobChangeController @0x647150 (GMS_v79_1_DEVM.exe,
// port 13340):
//
//	Decode1 @0x647163 — controlType (v2)
//	Decode4 @0x647166 — uniqueId (v3)
//	if controlType==0 -> sub_645DCC (reset, no tail)        @0x64718b
//	else Decode1 @0x647173 — aggro (v5), then sub_645CE1:
//	  Decode4 @0x645d06 — monsterId, CMob::SetTemporaryStat/Init — monster blob
//
// ActiveRequest(2) with aggro=true exercises the full tail. Monster blob omits
// the v87+ phase (model.go:512). Layout byte-identical to v83; no codec change.
//
// packet-audit:verify packet=monster/clientbound/MonsterControl version=gms_v79 ida=0x647150
func TestMonsterControlBytesV79(t *testing.T) {
	m := model.NewMonster(100, 200, 5, 300, model.MonsterAppearTypeRegen, 0)
	input := NewMonsterControl(ControlTypeActiveRequest, 5001, 100100, m, true)
	ctx := test.CreateContext("GMS", 79, 1)
	want := []byte{
		0x02,                   // controlType ActiveRequest — Decode1 @0x647163
		0x89, 0x13, 0x00, 0x00, // uniqueId 5001 — Decode4 @0x647166
		0x01,                   // aggro true — Decode1 @0x647173
		0x04, 0x87, 0x01, 0x00, // monsterId 100100 — Decode4 @0x645d06
		// --- monster blob (GMS>12 && <87) ---
		0x00, 0x00, 0x00, 0x00, // temp-stat mask H.hi (empty)
		0x00, 0x00, 0x00, 0x00, // mask H.lo
		0x00, 0x00, 0x00, 0x00, // mask L.hi
		0x00, 0x00, 0x00, 0x00, // mask L.lo
		0x64, 0x00, // x 100
		0xC8, 0x00, // y 200
		0x05,       // moveAction 5
		0x00, 0x00, // foothold 0
		0x2C, 0x01, // homeFoothold 300
		0xFE,       // appearType -2 (Regen)
		0x00,                   // team 0
		0x00, 0x00, 0x00, 0x00, // effectItemId 0
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v79 control bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMonsterControlAggroByteReflectsState pins the wire-level semantic from
// task-065 item 3: the aggro byte at position 5 (after controlType + mobId)
// reflects the real controller-aggro state instead of the legacy hardcoded
// byte(5).
func TestMonsterControlAggroByteReflectsState(t *testing.T) {
	m := model.NewMonster(100, 200, 5, 300, model.MonsterAppearTypeRegen, 0)
	ctx := test.CreateContext("GMS", 95, 1)
	cases := []struct {
		name     string
		aggro    bool
		wantByte byte
	}{
		{"aggro_true_emits_1", true, 1},
		{"aggro_false_emits_0", false, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			input := NewMonsterControl(ControlTypeActiveRequest, 5001, 100100, m, c.aggro)
			out := input.Encode(nil, ctx)(nil)
			// Wire layout: byte controlType + int32 uniqueId + byte aggro + ...
			// Aggro byte sits at index 5 (1 + 4 = 5).
			if len(out) < 6 {
				t.Fatalf("encoded packet too short: %d bytes", len(out))
			}
			if out[5] != c.wantByte {
				t.Errorf("aggro byte at index 5: got 0x%02x, want 0x%02x", out[5], c.wantByte)
			}
		})
	}
}
