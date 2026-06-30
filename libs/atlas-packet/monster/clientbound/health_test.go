package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=monster/clientbound/MonsterHealth version=gms_v83 ida=0x66d639
// packet-audit:verify packet=monster/clientbound/MonsterHealth version=gms_v87 ida=0x6a8505
// packet-audit:verify packet=monster/clientbound/MonsterHealth version=gms_v95 ida=0x642ef0
// packet-audit:verify packet=monster/clientbound/MonsterHealth version=jms_v185 ida=0x6eaddf
// packet-audit:verify packet=monster/clientbound/MonsterHealth version=gms_v84 ida=0x68393b
func TestMonsterHealth(t *testing.T) {
	input := NewMonsterHealth(5001, 85)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestMonsterHealthBytesV79 pins the exact wire bytes against the v79 client
// read order. uniqueId is consumed by CMobPool::OnMobPacket @0x646d46
// (Decode4 @0x646d50) before switching on op 228 -> CMob::OnHPIndicator
// @0x63c629 (GMS_v79_1_DEVM.exe, port 13340):
//
//	Decode1 @0x63c63c — hpPercent (stored at +324)
//
// Byte-identical to v83; no codec change.
//
// packet-audit:verify packet=monster/clientbound/MonsterHealth version=gms_v79 ida=0x63c629
func TestMonsterHealthBytesV79(t *testing.T) {
	input := NewMonsterHealth(5001, 85)
	ctx := test.CreateContext("GMS", 79, 1)
	want := []byte{
		0x89, 0x13, 0x00, 0x00, // uniqueId 5001 — pool Decode4 @0x646d50
		0x55, // hpPercent 85 — Decode1 @0x63c63c
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v79 health bytes:\n got % x\nwant % x", got, want)
	}
}
