package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

// packet-audit:verify packet=monster/clientbound/MonsterDestroy version=gms_v83 ida=0x67961d
// packet-audit:verify packet=monster/clientbound/MonsterDestroy version=gms_v87 ida=0x6b5169
// packet-audit:verify packet=monster/clientbound/MonsterDestroy version=gms_v95 ida=0x658b90
// packet-audit:verify packet=monster/clientbound/MonsterDestroy version=jms_v185 ida=0x6f8a1f
// packet-audit:verify packet=monster/clientbound/MonsterDestroy version=gms_v84 ida=0x6901b3
func TestMonsterDestroy(t *testing.T) {
	input := NewMonsterDestroy(5001, DestroyTypeFadeOut)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}

// TestMonsterDestroyBytesV79 pins the exact wire bytes against the v79 client
// read order in CMobPool::OnMobLeaveField @0x646ff6 (GMS_v79_1_DEVM.exe, port
// 13340):
//
//	Decode4 @0x647012 — uniqueId (v10)
//	Decode1 @0x64701a — destroyType (v3): non-zero -> fade (sub_647329),
//	                    zero -> death path
//
// v79 reads ONLY uniqueId(4)+destroyType(1); there is no destroyType==4 swallow
// arm and no trailing swallowCharacterId read (that path is v95+). The codec's
// swallow branch is gated on destroyType==4, which v79 never emits, so the
// standard FadeOut(1) shape is byte-identical to v83.
//
// packet-audit:verify packet=monster/clientbound/MonsterDestroy version=gms_v79 ida=0x646ff6
func TestMonsterDestroyBytesV79(t *testing.T) {
	input := NewMonsterDestroy(5001, DestroyTypeFadeOut)
	ctx := test.CreateContext("GMS", 79, 1)
	want := []byte{
		0x89, 0x13, 0x00, 0x00, // uniqueId 5001 — Decode4 @0x647012
		0x01, // destroyType FadeOut — Decode1 @0x64701a
	}
	got := input.Encode(nil, ctx)(nil)
	if !bytes.Equal(got, want) {
		t.Errorf("v79 destroy bytes:\n got % x\nwant % x", got, want)
	}
}

func TestMonsterDestroyBySwallow(t *testing.T) {
	input := NewMonsterDestroyBySwallow(5001, 12345)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
	// Confirm the trailing swallowCharacterId is present in the encoded bytes.
	// Wire shape for swallow: uint32(uniqueId)+byte(destroyType=4)+uint32(charId) = 9 bytes.
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 95, 1)
	bytes := input.Encode(l, ctx)(nil)
	if len(bytes) != 9 {
		t.Errorf("swallow encode: got %d bytes, want 9 (uint32 uid + byte type + uint32 swallowCharId)", len(bytes))
	}
	// Regression check: plain destroy stays at 5 bytes.
	plain := NewMonsterDestroy(5001, DestroyTypeFadeOut)
	plainBytes := plain.Encode(l, ctx)(nil)
	if len(plainBytes) != 5 {
		t.Errorf("plain destroy encode: got %d bytes, want 5 (uint32 uid + byte type)", len(plainBytes))
	}
}
