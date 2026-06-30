package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=monster/clientbound/MonsterMovement version=gms_v83 ida=0x66be61
// packet-audit:verify packet=monster/clientbound/MonsterMovement version=gms_v87 ida=0x6a6cb3
// packet-audit:verify packet=monster/clientbound/MonsterMovement version=gms_v95 ida=0x6521e0
// packet-audit:verify packet=monster/clientbound/MonsterMovement version=jms_v185 ida=0x6e955a
// packet-audit:verify packet=monster/clientbound/MonsterMovement version=gms_v84 ida=0x6820ea
func TestMonsterMovementRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			input := NewMonsterMovement(5001, false, true, false, 0, 0, 0, model.MultiTargetForBall{}, model.RandTimeForAreaAttack{}, model.Movement{})
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

func TestMonsterMovementRoundTripWithSkill(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			input := NewMonsterMovement(5001, true, true, true, 1, 100, 5, model.MultiTargetForBall{}, model.RandTimeForAreaAttack{}, model.Movement{})
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestMonsterMovementBytesV79 pins the exact wire bytes against the v79 client
// read order. uniqueId is consumed by the pool dispatcher
// CMobPool::OnMobPacket @0x646d46 (Decode4 @0x646d50) before it switches on
// op 217 -> CMob::OnMove @0x63a98b (GMS_v79_1_DEVM.exe, port 13340):
//
//	Decode1 @0x63a9a8 — bNotForceLandingWhenDiscard (v3)
//	Decode1 @0x63a9b5 — bNextAttackPossible (v4)
//	Decode1 @0x63a9ba — bLeft/action byte (v5)
//	Decode4 @0x63aa7a — skill word (skillId int16 + skillLevel int16)
//	CMovePath::OnMovePacket @0x63ad0f — opaque movement trailer
//
// v79 (<87) omits bNotChangeAction and the multiTargets/randTimeForAreaAttack
// blocks (movement.go MajorAtLeast(87) gate). Empty model.Movement encodes as
// StartX(2)+StartY(2)+count(1) = 5 zero bytes. Byte-identical to v83; no codec
// change.
//
// packet-audit:verify packet=monster/clientbound/MonsterMovement version=gms_v79 ida=0x63a98b
func TestMonsterMovementBytesV79(t *testing.T) {
	input := NewMonsterMovement(5001, true, true, true, 1, 100, 5, model.MultiTargetForBall{}, model.RandTimeForAreaAttack{}, model.Movement{})
	ctx := test.CreateContext("GMS", 79, 1)
	want := []byte{
		0x89, 0x13, 0x00, 0x00, // uniqueId 5001 — pool Decode4 @0x646d50
		0x01,       // bNotForceLandingWhenDiscard — Decode1 @0x63a9a8
		0x01,       // bNextAttackPossible — Decode1 @0x63a9b5
		0x01,       // bLeft 1 — Decode1 @0x63a9ba
		0x64, 0x00, // skillId 100 int16 — first half of Decode4 @0x63aa7a
		0x05, 0x00, // skillLevel 5 int16 — second half
		0x00, 0x00, // movement StartX — OnMovePacket @0x63ad0f
		0x00, 0x00, // movement StartY
		0x00, // movement element count = 0
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v79 movement bytes:\n got % x\nwant % x", got, want)
	}
}
