package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=monster/clientbound/MonsterMovementAck version=gms_v83 ida=0x66c23b
// packet-audit:verify packet=monster/clientbound/MonsterMovementAck version=gms_v87 ida=0x6a7106
// packet-audit:verify packet=monster/clientbound/MonsterMovementAck version=gms_v95 ida=0x640c50
// packet-audit:verify packet=monster/clientbound/MonsterMovementAck version=jms_v185 ida=0x6e99c8
// packet-audit:verify packet=monster/clientbound/MonsterMovementAck version=gms_v84 ida=0x68253d
func TestMonsterMovementAck(t *testing.T) {
	input := NewMonsterMovementAck(5001, 42, 300, true, 10, 3)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestMonsterMovementAckBytesV79 pins the exact wire bytes against the v79
// client read order. uniqueId is consumed by CMobPool::OnMobPacket @0x646d46
// (Decode4 @0x646d50) before switching on op 218 -> CMob::OnCtrlAck @0x63ad65
// (GMS_v79_1_DEVM.exe, port 13340):
//
//	Decode2 @0x63ad8a — moveId (v10, int16)
//	Decode1 @0x63ad95 — useSkills (v4)
//	Decode2 @0x63ad9f — mp (v5, uint16)
//	Decode1 @0x63adbe — skillId (v6)
//	Decode1 @0x63adc0 — skillLevel (v7)
//
// Byte-identical to v83; no codec change.
//
// packet-audit:verify packet=monster/clientbound/MonsterMovementAck version=gms_v79 ida=0x63ad65
func TestMonsterMovementAckBytesV79(t *testing.T) {
	input := NewMonsterMovementAck(5001, 42, 300, true, 10, 3)
	ctx := test.CreateContext("GMS", 79, 1)
	want := []byte{
		0x89, 0x13, 0x00, 0x00, // uniqueId 5001 — pool Decode4 @0x646d50
		0x2A, 0x00, // moveId 42 — Decode2 @0x63ad8a
		0x01,       // useSkills true — Decode1 @0x63ad95
		0x2C, 0x01, // mp 300 — Decode2 @0x63ad9f
		0x0A, // skillId 10 — Decode1 @0x63adbe
		0x03, // skillLevel 3 — Decode1 @0x63adc0
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v79 movement-ack bytes:\n got % x\nwant % x", got, want)
	}
}

// TestMonsterMovementAckBytesV72 pins the v72 wire. uniqueId via
// CMobPool::OnMobPacket @0x62560d (Decode4 @0x625617), op 212 -> CMob::OnCtrlAck
// @0x61b4d8 (GMS_v72.1_U_DEVM.exe, port 13339):
//
//	Decode2 @0x61b4fd — moveId (v10, int16)
//	Decode1 @0x61b508 — useSkills (v4)
//	Decode2 @0x61b512 — mp (v5, uint16)
//	Decode1 @0x61b531 — skillId (v6)
//	Decode1 @0x61b533 — skillLevel (v7)
//
// Byte-identical to v79; no codec change.
//
// packet-audit:verify packet=monster/clientbound/MonsterMovementAck version=gms_v72 ida=0x61b4d8
func TestMonsterMovementAckBytesV72(t *testing.T) {
	input := NewMonsterMovementAck(5001, 42, 300, true, 10, 3)
	ctx := test.CreateContext("GMS", 72, 1)
	want := []byte{
		0x89, 0x13, 0x00, 0x00, // uniqueId 5001 — pool Decode4 @0x625617
		0x2A, 0x00, // moveId 42 — Decode2 @0x61b4fd
		0x01,       // useSkills true — Decode1 @0x61b508
		0x2C, 0x01, // mp 300 — Decode2 @0x61b512
		0x0A, // skillId 10 — Decode1 @0x61b531
		0x03, // skillLevel 3 — Decode1 @0x61b533
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v72 movement-ack bytes:\n got % x\nwant % x", got, want)
	}
}
