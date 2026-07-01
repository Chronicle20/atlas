package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/BuffCancelForeign version=gms_v83 ida=0x983921
// packet-audit:verify packet=character/clientbound/BuffCancelForeign version=gms_v87 ida=0xa093ab
// packet-audit:verify packet=character/clientbound/BuffCancelForeign version=gms_v95 ida=0x953e40
// packet-audit:verify packet=character/clientbound/BuffCancel version=gms_v72 ida=0x918f3c
// packet-audit:verify packet=character/clientbound/BuffCancel version=gms_v79 ida=0x96ab32
// packet-audit:verify packet=character/clientbound/BuffCancel version=gms_v83 ida=0xa2071f
// packet-audit:verify packet=character/clientbound/BuffCancel version=gms_v87 ida=0xab7dc1
// packet-audit:verify packet=character/clientbound/BuffCancel version=gms_v95 ida=0x9f2ab0
// packet-audit:verify packet=character/clientbound/BuffCancelForeign version=gms_v84 ida=0x9c3cbf
// packet-audit:verify packet=character/clientbound/BuffCancel version=gms_v84 ida=0xa6bb24
// packet-audit:verify packet=character/clientbound/BuffCancel version=jms_v185 ida=0xb07628
// packet-audit:verify packet=character/clientbound/BuffCancelForeign version=jms_v185 ida=0xa574f5
func TestBuffCancelRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			cts := model.NewCharacterTemporaryStat()
			input := NewBuffCancel(*cts)
			output := BuffCancel{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

func TestBuffCancelForeignRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			cts := model.NewCharacterTemporaryStat()
			input := NewBuffCancelForeign(99999, *cts)
			output := BuffCancelForeign{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.CharacterId() != 99999 {
				t.Errorf("characterId: got %v, want %v", output.CharacterId(), 99999)
			}
		})
	}
}

// TestBuffCancelV79ByteFixture pins the v79 empty-CTS wire: the 16-byte
// SecondaryStat reset mask followed by the trailing tSwallowBuffTime byte. The v79
// client (CWvsContext::OnTemporaryStatReset @0x96ab32) reads the mask via
// DecodeBuffer(16) and reads the trailing Decode1 only when the mask carries a
// movement-affecting stat (none here) — Atlas writes it unconditionally
// (harmless over-write). The v79 CTS registry path is byte-identical to v83, so the
// empty mask is 00000000 0001FC00 (bits 82-88, two-state base group) — see
// v79EmptyMask in buff_give_test.go (§5 opaque caveat).
// TestBuffCancelV72ByteFixture pins the legacy GMS v72 empty-CTS reset wire. v72 < 87
// so the CTS model's version gates (87 / 95) do not fire — the 16-byte reset mask is
// byte-identical to v79 (v79EmptyMask, bits 82-88). IDA-verified: CWvsContext::
// OnTemporaryStatReset @0x918f3c (GMS_v72.1_U_DEVM.exe, port 13339) reads the mask via
// DecodeBuffer(16) into a UINT128, then reads the trailing Decode1 only when the mask
// carries a movement-affecting stat (none here) — Atlas writes it unconditionally
// (harmless over-write). 17 bytes total, same structure as v79 (§5 opaque caveat).
func TestBuffCancelV72ByteFixture(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := NewBuffCancel(*model.NewCharacterTemporaryStat()).Encode(nil, ctx)(nil)
	if !bytes.Equal(got[:16], v79EmptyMask) {
		t.Errorf("v72 BuffCancel flag word: got %x want %x", got[:16], v79EmptyMask)
	}
	if len(got) != 17 {
		t.Fatalf("v72 BuffCancel length: got %d want 17 (16 mask + 1 trailer)", len(got))
	}
	if got[16] != 0x00 {
		t.Errorf("v72 BuffCancel trailer byte: got %02x want 00", got[16])
	}
}

func TestBuffCancelV79ByteFixture(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := NewBuffCancel(*model.NewCharacterTemporaryStat()).Encode(nil, ctx)(nil)
	if !bytes.Equal(got[:16], v79EmptyMask) {
		t.Errorf("v79 BuffCancel flag word: got %x want %x", got[:16], v79EmptyMask)
	}
	// EncodeMask(16) + WriteByte(0) tSwallowBuffTime → 17 bytes total.
	if len(got) != 17 {
		t.Fatalf("v79 BuffCancel length: got %d want 17 (16 mask + 1 trailer)", len(got))
	}
	if got[16] != 0x00 {
		t.Errorf("v79 BuffCancel trailer byte: got %02x want 00", got[16])
	}
}
