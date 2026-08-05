package clientbound

import (
	"bytes"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

// TestBuffCancelV79ByteFixture pins the v79 empty-CTS wire: a 16-byte
// SecondaryStat CANCEL mask (task-167 F1: CancelMask, not the give-shape
// EncodeMask — an empty CTS carries none of the unconditional two-state
// group bits) and NO trailing movement byte. The v79 client
// (CWvsContext::OnTemporaryStatReset @0x96ab32) reads the mask via
// DecodeBuffer(16) and reads the trailing Decode1 only when the mask carries
// a movement-affecting stat — none here, since the cancel mask is all-zero.
// TestBuffCancelV72ByteFixture pins the legacy GMS v72 empty-CTS reset wire. v72 < 87
// so the CTS model's version gates (87 / 95) do not fire — the 16-byte cancel mask is
// byte-identical to v79 (all-zero, task-167 F1). IDA-verified: CWvsContext::
// OnTemporaryStatReset @0x918f3c (GMS_v72.1_U_DEVM.exe, port 13339) reads the mask via
// DecodeBuffer(16) into a UINT128, then reads the trailing Decode1 only when the mask
// carries a movement-affecting stat (none here). 16 bytes total, same structure as v79.
func TestBuffCancelV72ByteFixture(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := NewBuffCancel(*model.NewCharacterTemporaryStat()).Encode(nil, ctx)(nil)
	want := make([]byte, 16)
	if !bytes.Equal(got, want) {
		t.Errorf("v72 BuffCancel empty-CTS wire: got %x want %x (16-byte zero cancel mask, no movement byte)", got, want)
	}
}

func TestBuffCancelV79ByteFixture(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	got := NewBuffCancel(*model.NewCharacterTemporaryStat()).Encode(nil, ctx)(nil)
	want := make([]byte, 16)
	if !bytes.Equal(got, want) {
		t.Errorf("v79 BuffCancel empty-CTS wire: got %x want %x (16-byte zero cancel mask, no movement byte)", got, want)
	}
}

// TestBuffCancelV61ByteFixture pins the very-legacy GMS v61 empty-CTS reset wire. v61
// < 87 so the CTS model's version gates (87 / 95) do not fire — the 16-byte cancel mask
// is byte-identical to v72/v79 (all-zero, task-167 F1). IDA-verified: the real
// per-op handler CWvsContext::OnTemporaryStatReset @0x84353a (GMS_v61.1_U_DEVM.exe, port
// 13338) reads the mask via DecodeBuffer(16) @0x843560 into a UINT128, then reads a
// trailing Decode1 @0x84365f only when the mask carries a movement-affecting stat (none
// here). 16 bytes total, same structure as v72.
// packet-audit:verify packet=character/clientbound/BuffCancel version=gms_v61 ida=0x84353a
func TestBuffCancelV61ByteFixture(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := NewBuffCancel(*model.NewCharacterTemporaryStat()).Encode(nil, ctx)(nil)
	want := make([]byte, 16)
	if !bytes.Equal(got, want) {
		t.Errorf("v61 BuffCancel empty-CTS wire: got %x want %x (16-byte zero cancel mask, no movement byte)", got, want)
	}
}

// TestBuffCancelV48ByteFixture pins the very-legacy GMS v48 empty-CTS reset wire.
// Pre-v61 the SecondaryStat mask is a plain 8-byte little-endian value (NOT the
// 128-bit UINT128), read by CWvsContext::OnTemporaryStatReset @0x71b054 via
// CInPacket::DecodeBuffer(&v8, 8) @0x71b06e (GMS_v48_1_DEVM.exe, port 13337). An
// empty CTS's cancel mask is all-zero (task-167 F1), and the trailing Decode1 is
// read only when the mask carries a movement stat (none here), so the wire is
// exactly the 8-byte zero mask with no trailing byte.
// packet-audit:verify packet=character/clientbound/BuffCancel version=gms_v48 ida=0x71b054
func TestBuffCancelV48ByteFixture(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	got := NewBuffCancel(*model.NewCharacterTemporaryStat()).Encode(nil, ctx)(nil)
	want := []byte{0, 0, 0, 0, 0, 0, 0, 0} // 8-byte zero mask, no movement byte
	if !bytes.Equal(got, want) {
		t.Errorf("v48 BuffCancel wire: got %x want %x", got, want)
	}
}

// Beacon-only cancel: mask carries exactly the GuidedBullet bit (v83 shift 87
// -> wire dword[1] 0x00800000) and NO movement byte (GuidedBullet is not in
// the movement filter; the client reads the trailing byte only when
// sub_77DC78(mask) is true — design.md §2.3).
func TestBuffCancelBeaconOnlyV83(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	cts := model.NewCharacterTemporaryStat()
	cts.AddStat(nil)(tn)(string(character.TemporaryStatTypeHomingBeacon), 5211006, 1000001, 1, time.Time{})

	got := NewBuffCancel(*cts).Encode(nil, ctx)(nil)

	want := []byte{
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x80, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("beacon-only cancel: got % x want % x (16 bytes, no movement byte)", got, want)
	}
}

// A movement-affecting cancel (Speed) carries the trailing byte.
func TestBuffCancelSpeedCarriesMovementByteV83(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	cts := model.NewCharacterTemporaryStat()
	cts.AddStat(nil)(tn)(string(character.TemporaryStatTypeSpeed), 2001002, 20, 10, time.Now().Add(time.Minute))

	got := NewBuffCancel(*cts).Encode(nil, ctx)(nil)

	if len(got) != 17 {
		t.Fatalf("speed cancel length: got %d want 17 (mask + movement byte)", len(got))
	}
	// Speed is registry shift 7 -> mask.L low dword -> wire dword[3]
	// (bytes 12-15) = 80 00 00 00; dwords [0..2] empty.
	if !bytes.Equal(got[0:12], make([]byte, 12)) {
		t.Fatalf("speed cancel mask dwords[0..2] should be empty: got % x", got[0:12])
	}
	if !bytes.Equal(got[12:16], []byte{0x80, 0x00, 0x00, 0x00}) {
		t.Fatalf("speed cancel mask dword[3]: got % x", got[12:16])
	}
}

// A mount cancel carries the riding bit and the movement byte, and nothing else.
func TestBuffCancelMountV83(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	cts := model.NewCharacterTemporaryStat()
	cts.AddStat(nil)(tn)(string(character.TemporaryStatTypeMonsterRiding), 1004, 1902000, 1, time.Now().Add(time.Hour))

	got := NewBuffCancel(*cts).Encode(nil, ctx)(nil)

	if len(got) != 17 {
		t.Fatalf("mount cancel length: got %d want 17", len(got))
	}
	// RideVehicle shift 85 -> dword[1] = 0x00200000 -> LE 00 00 20 00.
	if !bytes.Equal(got[4:8], []byte{0x00, 0x00, 0x20, 0x00}) {
		t.Fatalf("mount cancel mask dword[1]: got % x want 00 00 20 00", got[4:8])
	}
}

// v95 beacon-only cancel: exactly bit 127 (dword[0] 0x80000000), no movement byte.
func TestBuffCancelBeaconOnlyV95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 95, 1)
	cts := model.NewCharacterTemporaryStat()
	cts.AddStat(nil)(tn)(string(character.TemporaryStatTypeHomingBeacon), 5220011, 1000001, 10, time.Time{})

	got := NewBuffCancel(*cts).Encode(nil, ctx)(nil)

	want := append([]byte{0x00, 0x00, 0x00, 0x80}, make([]byte, 12)...)
	if !bytes.Equal(got, want) {
		t.Fatalf("v95 beacon-only cancel: got % x want % x", got, want)
	}
}
