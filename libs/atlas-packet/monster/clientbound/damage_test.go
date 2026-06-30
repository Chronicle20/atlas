package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=monster/clientbound/MonsterDamage version=gms_v83 ida=0x66c6c2
// packet-audit:verify packet=monster/clientbound/MonsterDamage version=gms_v87 ida=0x6a758d
// packet-audit:verify packet=monster/clientbound/MonsterDamage version=gms_v95 ida=0x64ecb0
// packet-audit:verify packet=monster/clientbound/MonsterDamage version=jms_v185 ida=0x6e9e43
// packet-audit:verify packet=monster/clientbound/MonsterDamage version=gms_v84 ida=0x6829c4
func TestMonsterDamage(t *testing.T) {
	input := NewMonsterDamage(5001, MonsterDamageTypeUnk2, 1500, 8500, 10000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestMonsterDamageBytesV79 pins the exact wire bytes against the v79 client
// read order. uniqueId is consumed by CMobPool::OnMobPacket @0x646d46
// (Decode4 @0x646d50) before switching on op 224 -> CMob::OnDamaged @0x63b6b2
// (GMS_v79_1_DEVM.exe, port 13340):
//
//	Decode1 @0x63b6bd — damageType
//	Decode4 @0x63b6cd — damage (v3)
//	Decode4 @0x63b6f9 — hp (v4)     ] read only when the mob carries the
//	Decode4 @0x63b6fb — maxHp (v5)  ] HP-gauge flag (template +520); the server
//	                                  always emits the pair and a gauge mob
//	                                  consumes it. Codec writes both
//	                                  unconditionally, matching v83/87/95/jms.
//
// Byte-identical to v83; no codec change.
//
// packet-audit:verify packet=monster/clientbound/MonsterDamage version=gms_v79 ida=0x63b6b2
func TestMonsterDamageBytesV79(t *testing.T) {
	input := NewMonsterDamage(5001, MonsterDamageTypeUnk2, 1500, 8500, 10000)
	ctx := test.CreateContext("GMS", 79, 1)
	want := []byte{
		0x89, 0x13, 0x00, 0x00, // uniqueId 5001 — pool Decode4 @0x646d50
		0x01,                   // damageType Unk2 — Decode1 @0x63b6bd
		0xDC, 0x05, 0x00, 0x00, // damage 1500 — Decode4 @0x63b6cd
		0x34, 0x21, 0x00, 0x00, // hp 8500 — Decode4 @0x63b6f9
		0x10, 0x27, 0x00, 0x00, // maxHp 10000 — Decode4 @0x63b6fb
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v79 damage bytes:\n got % x\nwant % x", got, want)
	}
}
