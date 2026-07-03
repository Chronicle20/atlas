package clientbound

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/CharacterSkillChange version=gms_v72 ida=0x9174dd
// packet-audit:verify packet=character/clientbound/CharacterSkillChange version=gms_v79 ida=0x968f0e
// packet-audit:verify packet=character/clientbound/CharacterSkillChange version=gms_v83 ida=0xa1e48c
// packet-audit:verify packet=character/clientbound/CharacterSkillChange version=gms_v87 ida=0xab57c5
// packet-audit:verify packet=character/clientbound/CharacterSkillChange version=gms_v95 ida=0x9f5f30
// packet-audit:verify packet=character/clientbound/CharacterSkillChange version=jms_v185 ida=0xb04ff3
// packet-audit:verify packet=character/clientbound/CharacterSkillChange version=gms_v84 ida=0xa6972b
func TestCharacterSkillChange(t *testing.T) {
	input := NewCharacterSkillChange(true, 1001003, 10, 0, time.Time{}, false)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
	// v79 legacy: no per-skill expiration field on the wire.
	t.Run("GMS v79 roundtrip", func(t *testing.T) {
		ctx := test.CreateContext("GMS", 79, 1)
		out := CharacterSkillChange{}
		test.RoundTrip(t, ctx, input.Encode, out.Decode, nil)
	})
}

// TestCharacterSkillChangeV79ByteFixture pins the legacy GMS v79 wire, which OMITS
// the 8-byte per-skill expiration field. CWvsContext::OnChangeSkillRecordResult
// @0x968f0e reads, per skill: Decode4 skillId, Decode4 level, Decode4 masterLevel
// (3 ints, no DecodeBuffer), then a trailing Decode1 (sn) after the loop — versus
// v83 @0xa1e48c which additionally does DecodeBuffer(8) for the expiration. The
// encoder gates the int64 off for GMS <83, so the total body is 16 bytes (not 24).
// Verified against CWvsContext::OnChangeSkillRecordResult @0x968f0e (marker pinned
// on TestCharacterSkillChange above).
// TestCharacterSkillChangeV72ByteFixture pins the legacy GMS v72 wire. IDA-verified
// CWvsContext::OnChangeSkillRecordResult @0x9174dd (GMS_v72.1_U_DEVM.exe, port 13339)
// reads: Decode1 exclRequestSent @0x9174eb, Decode2 count @0x917526, then per skill
// Decode4 skillId @0x917544 + Decode4 level @0x91754e + Decode4 masterLevel @0x91757f
// (3 ints, NO DecodeBuffer), then a trailing Decode1 sn @0x9175cb after the loop —
// byte-identical to v79 (the 8-byte expiration was added at v83). 16-byte body.
func TestCharacterSkillChangeV72ByteFixture(t *testing.T) {
	ctx := test.CreateContext("GMS", 72, 1)
	input := NewCharacterSkillChange(true, 1001003, 10, 0, time.Time{}, false)
	expected := []byte{
		0x01,       // exclRequestSent           @0x9174eb
		0x01, 0x00, // count=1                    @0x917526
		0x2B, 0x46, 0x0F, 0x00, // skillId=1001003 LE   @0x917544
		0x0A, 0x00, 0x00, 0x00, // level=10 LE          @0x91754e
		0x00, 0x00, 0x00, 0x00, // masterLevel=0 LE     @0x91757f
		0x00, // sn=false (no expiration on v72) @0x9175cb
	}
	got := test.Encode(t, ctx, input.Encode, nil)
	if len(got) != len(expected) {
		t.Fatalf("byte length mismatch: got %d want %d\n  got:  %X\n  want: %X", len(got), len(expected), got, expected)
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("byte[%d] = %02X, want %02X\n  got:  %X\n  want: %X", i, got[i], expected[i], got, expected)
			break
		}
	}
}

// TestCharacterSkillChangeV61ByteFixture pins the very-legacy GMS v61 wire, which OMITS
// the 8-byte per-skill expiration field (added at v83). IDA-verified: the real per-op
// handler CWvsContext::OnChangeSkillRecordResult @0x841d56 (GMS_v61.1_U_DEVM.exe, port
// 13338 — registry's dispatcher note-address 0x830d6b is the switch, not the handler)
// reads Decode1 exclRequestSent @0x841d64, Decode2 count @0x841d9f, then per skill
// Decode4 skillId @0x841dbd + Decode4 level @0x841dc7 + Decode4 masterLevel @0x841df8
// (3 ints, NO DecodeBuffer), then a trailing Decode1 sn @0x841e44 after the loop —
// byte-identical to v72 (the encoder gates the int64 expiration off for GMS <83). 16-byte body.
// packet-audit:verify packet=character/clientbound/CharacterSkillChange version=gms_v61 ida=0x841d56
func TestCharacterSkillChangeV61ByteFixture(t *testing.T) {
	ctx := test.CreateContext("GMS", 61, 1)
	input := NewCharacterSkillChange(true, 1001003, 10, 0, time.Time{}, false)
	expected := []byte{
		0x01,       // exclRequestSent           @0x841d64
		0x01, 0x00, // count=1                    @0x841d9f
		0x2B, 0x46, 0x0F, 0x00, // skillId=1001003 LE   @0x841dbd
		0x0A, 0x00, 0x00, 0x00, // level=10 LE          @0x841dc7
		0x00, 0x00, 0x00, 0x00, // masterLevel=0 LE     @0x841df8
		0x00, // sn=false (no expiration on v61) @0x841e44
	}
	got := test.Encode(t, ctx, input.Encode, nil)
	if len(got) != len(expected) {
		t.Fatalf("byte length mismatch: got %d want %d\n  got:  %X\n  want: %X", len(got), len(expected), got, expected)
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("byte[%d] = %02X, want %02X\n  got:  %X\n  want: %X", i, got[i], expected[i], got, expected)
			break
		}
	}
}

func TestCharacterSkillChangeV79ByteFixture(t *testing.T) {
	ctx := test.CreateContext("GMS", 79, 1)
	input := NewCharacterSkillChange(true, 1001003, 10, 0, time.Time{}, false)
	// exclRequestSent=true (0x01)
	// count=1 (0x0001 LE = 01 00)
	// skillId=1001003 (0x000F462B LE = 2B 46 0F 00)
	// level=10 (0x0000000A LE = 0A 00 00 00)
	// masterLevel=0 (00 00 00 00)
	// -- no int64 expiration on v79 --
	// sn=false (0x00)
	expected := []byte{
		0x01,       // exclRequestSent
		0x01, 0x00, // count=1
		0x2B, 0x46, 0x0F, 0x00, // skillId=1001003 LE
		0x0A, 0x00, 0x00, 0x00, // level=10 LE
		0x00, 0x00, 0x00, 0x00, // masterLevel=0 LE
		0x00, // sn=false
	}
	got := test.Encode(t, ctx, input.Encode, nil)
	if len(got) != len(expected) {
		t.Fatalf("byte length mismatch: got %d want %d\n  got:  %X\n  want: %X", len(got), len(expected), got, expected)
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("byte[%d] = %02X, want %02X\n  got:  %X\n  want: %X", i, got[i], expected[i], got, expected)
			break
		}
	}
}
