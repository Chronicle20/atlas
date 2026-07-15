package clientbound

import (
	"bytes"
	"testing"

	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// TestGuideTalkIdxByteOutputV79 pins the gms_v79 TALK_GUIDE (op 0x0D1)
// clientbound idx-arm wire. IDA-verified client decode (GMS_v79_1_DEVM.exe,
// port 13340) — CUserLocal::OnTutorMsg @0x8b818a:
//
//	if ( CInPacket::Decode1(a2) )      @0x8b81a7 → bByMessage byte; nonzero = idx arm.
//	  v4 = CInPacket::Decode4(v3)      @0x8b81b9 → hintId uint32-LE.
//	  v5 = CInPacket::Decode4(v3)      @0x8b81bb → duration uint32-LE.
//
// (The bByMessage==0 arm is DecodeStr(message)+Decode4+Decode4 = GuideTalkMessage.)
// Version-stable bool-gated shape; matches the codec.
//
// packet-audit:verify packet=npc/clientbound/NpcGuideTalkIdx version=gms_v79 ida=0x8b818a
// packet-audit:verify packet=npc/clientbound/NpcGuideTalkMessage version=gms_v79 ida=0x8b818a
func TestGuideTalkIdxByteOutputV79(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := test.CreateContext("GMS", 79, 1)
	// idx arm: bByMessage(0x01) + hintId=5 + duration=7000(0x1B58).
	input := NewGuideTalkIdx(5, 7000)
	expected := []byte{
		0x01,                   // bByMessage != 0 (idx arm)
		0x05, 0x00, 0x00, 0x00, // hintId = 5
		0x58, 0x1B, 0x00, 0x00, // duration = 7000
	}
	if got := input.Encode(l, ctx)(nil); !bytes.Equal(got, expected) {
		t.Errorf("v79 guide-talk idx golden mismatch: got %v want %v", got, expected)
	}

	// message arm: bByMessage(0x00) + Str("hi") + width + duration.
	msg := NewGuideTalkMessage("hi", 200, 4000)
	wantMsg := []byte{
		0x00,             // bByMessage == 0 (message arm)
		0x02, 0x00, 'h', 'i', // message "hi"
		0xC8, 0x00, 0x00, 0x00, // width = 200
		0xA0, 0x0F, 0x00, 0x00, // duration = 4000
	}
	if got := msg.Encode(l, ctx)(nil); !bytes.Equal(got, wantMsg) {
		t.Errorf("v79 guide-talk message golden mismatch: got %v want %v", got, wantMsg)
	}
}

// TestGuideTalkMessage exercises the round-trip across all tenant variants and
// pins the leading branch byte. Per the v95 client (CUserLocal::OnTutorMsg
// @0x916f60) the string/message arm is bByMessage==0, so the first wire byte
// MUST be 0x00.
// packet-audit:verify packet=npc/clientbound/NpcGuideTalkMessage version=gms_v95 ida=0x916f60
// packet-audit:verify packet=npc/clientbound/NpcGuideTalkIdx version=gms_v83 ida=0x960239
// packet-audit:verify packet=npc/clientbound/NpcGuideTalkIdx version=gms_v87 ida=0x9e36c9
// packet-audit:verify packet=npc/clientbound/NpcGuideTalkIdx version=jms_v185 ida=0xa2d342
// packet-audit:verify packet=npc/clientbound/NpcGuideTalkMessage version=gms_v83 ida=0x960239
// packet-audit:verify packet=npc/clientbound/NpcGuideTalkMessage version=gms_v87 ida=0x9e36c9
// packet-audit:verify packet=npc/clientbound/NpcGuideTalkMessage version=jms_v185 ida=0xa2d342
// packet-audit:verify packet=npc/clientbound/NpcGuideTalkMessage version=gms_v84 ida=0x99f28c
// packet-audit:verify packet=npc/clientbound/NpcGuideTalkIdx version=gms_v84 ida=0x99f28c
func TestGuideTalkMessage(t *testing.T) {
	input := NewGuideTalkMessage("Hello adventurer!", 200, 4000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)

			l, _ := testlog.NewNullLogger()
			b := input.Encode(l, ctx)(nil)
			if len(b) == 0 || b[0] != 0x00 {
				t.Fatalf("GuideTalkMessage leading byte: got %#v, want 0x00 (message arm)", b)
			}
		})
	}
}

// TestGuideTalkIdx exercises the round-trip across all tenant variants and pins
// the leading branch byte. Per the v95 client (CUserLocal::OnTutorMsg
// @0x916f60) the hint-index arm is bByMessage!=0, so the first wire byte MUST
// be 0x01.
// packet-audit:verify packet=npc/clientbound/NpcGuideTalkIdx version=gms_v95 ida=0x916f60
func TestGuideTalkIdx(t *testing.T) {
	input := NewGuideTalkIdx(5, 7000)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)

			l, _ := testlog.NewNullLogger()
			b := input.Encode(l, ctx)(nil)
			if len(b) == 0 || b[0] != 0x01 {
				t.Fatalf("GuideTalkIdx leading byte: got %#v, want 0x01 (index arm)", b)
			}
		})
	}
}
