package clientbound

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

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
