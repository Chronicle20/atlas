package clientbound

import (
	"bytes"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

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

// emptyResetMask is the 16-byte SecondaryStat mask for a reset that cancels
// nothing. It is ALL ZERO — deliberately NOT v79EmptyMask (bits 82-88, the
// two-state base group), which these fixtures previously reused from
// buff_give_test.go.
//
// A SET must assert those base bits: the client reads one base-stat block per
// set bit, so dropping one desyncs the tail. A RESET carries no blocks at all,
// so each set bit is a bare instruction to tear that stat down. Reusing the give
// constant here meant every buff cancel — a mob's Slow expiring, say — told the
// client to reset RideVehicle and GuidedBullet as well.
//
// That is not theoretical: CWvsContext::OnTemporaryStatReset branches on the
// received mask and calls CUser::ShowRideVehicleEffect / CMobPool::ResetGuidedMob,
// then SecondaryStat::Reset(mask). A player mounted on a Battleship had the ride
// torn down client-side on every unrelated debuff expiry while the server still
// believed them mounted (task-190). Verified structurally identical on GMS v61
// (@0x84353a), v72 (@0x918f3c), v83 (@0xa2071f) and v95 (@0x9f2ab0, whose
// PDB-backed symbols name the branch constants CTS_RideVehicle_2 /
// CTS_GuidedBullet_0 outright), and on the foreign path
// CUserRemote::OnResetTemporaryStat (v83 @0x983921).
//
// The previous expectation was a round-trip fixture pinned against Atlas's own
// encoder, never against observed client bytes — so it locked the bug in rather
// than catching it.
var emptyResetMask = make([]byte, 16)

// The three legacy fixtures below pin the empty-CTS reset wire: 16-byte mask,
// then the trailing nSecondaryStatChangedPoint byte. Each client reads the mask
// via DecodeBuffer(16) and reads the trailing Decode1 only when the mask carries
// a movement-affecting stat (none here) — Atlas writes it unconditionally, which
// is harmless slack when unread and mandatory when read. v61/v72/v79 are all < 87,
// so the CTS model's 87/95 version gates do not fire and all three masks are
// byte-identical. 17 bytes total (§5 opaque caveat).

func TestBuffCancelV72ByteFixture(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := NewBuffCancel(*model.NewCharacterTemporaryStat()).Encode(nil, ctx)(nil)
	if !bytes.Equal(got[:16], emptyResetMask) {
		t.Errorf("v72 BuffCancel flag word: got %x want %x", got[:16], emptyResetMask)
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
	if !bytes.Equal(got[:16], emptyResetMask) {
		t.Errorf("v79 BuffCancel flag word: got %x want %x", got[:16], emptyResetMask)
	}
	if len(got) != 17 {
		t.Fatalf("v79 BuffCancel length: got %d want 17 (16 mask + 1 trailer)", len(got))
	}
	if got[16] != 0x00 {
		t.Errorf("v79 BuffCancel trailer byte: got %02x want 00", got[16])
	}
}

// packet-audit:verify packet=character/clientbound/BuffCancel version=gms_v61 ida=0x84353a
func TestBuffCancelV61ByteFixture(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	got := NewBuffCancel(*model.NewCharacterTemporaryStat()).Encode(nil, ctx)(nil)
	if !bytes.Equal(got[:16], emptyResetMask) {
		t.Errorf("v61 BuffCancel flag word: got %x want %x", got[:16], emptyResetMask)
	}
	if len(got) != 17 {
		t.Fatalf("v61 BuffCancel length: got %d want 17 (16 mask + 1 trailer)", len(got))
	}
	if got[16] != 0x00 {
		t.Errorf("v61 BuffCancel trailer byte: got %02x want 00", got[16])
	}
}

// TestBuffCancelV48ByteFixture pins the very-legacy GMS v48 empty-CTS reset wire.
// Pre-v61 the SecondaryStat mask is a plain 8-byte little-endian value (NOT the
// 128-bit UINT128), read by CWvsContext::OnTemporaryStatReset @0x71b054 via
// CInPacket::DecodeBuffer(&v8, 8) @0x71b06e (GMS_v48_1_DEVM.exe, port 13337). An
// empty CTS's reset mask is all-zero, and the trailing Decode1 is read only when
// the mask carries a movement stat (none here) — Atlas writes it unconditionally,
// harmless slack when unread and mandatory when read.
// packet-audit:verify packet=character/clientbound/BuffCancel version=gms_v48 ida=0x71b054
func TestBuffCancelV48ByteFixture(t *testing.T) {
	ctx := pt.CreateContext("GMS", 48, 1)
	got := NewBuffCancel(*model.NewCharacterTemporaryStat()).Encode(nil, ctx)(nil)
	want := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0} // 8-byte zero mask + trailer
	if !bytes.Equal(got, want) {
		t.Errorf("v48 BuffCancel wire: got %x want %x", got, want)
	}
}

// rideVehicleBit is the v83 CTS_RideVehicle mask bit as it lands on the wire:
// byte 6, value 0x20. Read straight out of the client — the 16 bytes at
// GMS v83 dword_BF5548 are all zero except byte 6 == 0x20, and
// CWvsContext::OnTemporaryStatReset (@0xa2071f) calls
// CUser::ShowRideVehicleEffect exactly when (receivedMask & that constant) is
// non-zero. Asserting the wire byte rather than a registry lookup keeps the
// test honest: if the shift moves, this fails instead of following it.
const (
	rideVehicleMaskByte  = 6
	rideVehicleMaskValue = 0x20
)

// TestBuffCancelOmitsRideVehicleForUnrelatedStat is the task-190 regression.
// Cancelling a mob disease must not tell the client to dismount. Before the
// fix the mask carried the whole two-state base group unconditionally, so this
// byte was 0xfc and (0xfc & 0x20) tripped the ShowRideVehicleEffect branch on
// every debuff expiry.
func TestBuffCancelOmitsRideVehicleForUnrelatedStat(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	cts := model.NewCharacterTemporaryStat()
	cts.AddStat(logrus.New())(tenant.MustFromContext(ctx))("SLOW", 126, 80, 2, time.Now().Add(time.Minute))

	got := NewBuffCancel(*cts).Encode(nil, ctx)(nil)

	if got[rideVehicleMaskByte]&rideVehicleMaskValue != 0 {
		t.Errorf("SLOW cancel asserts the RideVehicle bit: mask byte %d = %02x, want bit %02x clear (client would run ShowRideVehicleEffect)",
			rideVehicleMaskByte, got[rideVehicleMaskByte], rideVehicleMaskValue)
	}
	if bytes.Equal(got[:16], emptyResetMask) {
		t.Error("SLOW cancel encoded an all-zero mask; the SLOW bit itself must still be set")
	}
}

// TestBuffCancelKeepsRideVehicleForMountCancel is the other half: a genuine
// dismount must still carry the bit, or the client never tears the ride down.
// Without this, "omit the base bits" could be satisfied by omitting them always.
func TestBuffCancelKeepsRideVehicleForMountCancel(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	cts := model.NewCharacterTemporaryStat()
	cts.AddStat(logrus.New())(tenant.MustFromContext(ctx))("MONSTER_RIDING", 5221006, 1932000, 10, time.Now().Add(time.Minute))

	got := NewBuffCancel(*cts).Encode(nil, ctx)(nil)

	if got[rideVehicleMaskByte]&rideVehicleMaskValue == 0 {
		t.Errorf("mount cancel dropped the RideVehicle bit: mask byte %d = %02x, want bit %02x set",
			rideVehicleMaskByte, got[rideVehicleMaskByte], rideVehicleMaskValue)
	}
}

// TestBuffGiveOmitsRideVehicleForUnrelatedStat is the give-side half of the
// task-190 regression, and the one that matters for the reported symptom:
// CWvsContext::OnTemporaryStatSet runs ShowRideVehicleEffect (or, on a
// ladder/rope, SendSkillCancelRequest for 5221006) whenever the SET mask carries
// CTS_RideVehicle. A mob disease landing on a mounted player must not do that.
func TestBuffGiveOmitsRideVehicleForUnrelatedStat(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	cts := model.NewCharacterTemporaryStat()
	cts.AddStat(logrus.New())(tenant.MustFromContext(ctx))("SLOW", 126, 80, 2, time.Now().Add(time.Minute))

	got := NewBuffGive(*cts).Encode(nil, ctx)(nil)

	if got[rideVehicleMaskByte]&rideVehicleMaskValue != 0 {
		t.Errorf("SLOW give asserts the RideVehicle bit: mask byte %d = %02x, want bit %02x clear",
			rideVehicleMaskByte, got[rideVehicleMaskByte], rideVehicleMaskValue)
	}
}

// TestBuffGiveKeepsRideVehicleForMount is the over-correction guard: a real
// mount give must still carry the bit AND its base-stat block, or the client
// never renders the ride. Bits and blocks are gated on the same presence test,
// so this also proves they did not drift apart.
func TestBuffGiveKeepsRideVehicleForMount(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	cts := model.NewCharacterTemporaryStat()
	cts.AddStat(logrus.New())(tenant.MustFromContext(ctx))("MONSTER_RIDING", 5221006, 1932000, 10, time.Now().Add(time.Minute))

	got := NewBuffGive(*cts).Encode(nil, ctx)(nil)

	if got[rideVehicleMaskByte]&rideVehicleMaskValue == 0 {
		t.Errorf("mount give dropped the RideVehicle bit: mask byte %d = %02x, want bit %02x set",
			rideVehicleMaskByte, got[rideVehicleMaskByte], rideVehicleMaskValue)
	}
	// 16 mask + 2 defense bytes + one 13-byte MonsterRiding base block + the
	// 3-byte give trailer. A dropped block would land at 21.
	if len(got) != 34 {
		t.Errorf("mount give length: got %d want 34 (mask+defense+MonsterRiding block+trailer)", len(got))
	}
}

// The three fixtures below are task-167's beacon/movement cancel shapes, kept
// through the task-190 merge with one change: the trailing
// nSecondaryStatChangedPoint byte is now written unconditionally, so a
// non-movement cancel is 17 bytes rather than 16. The mask contents — the part
// task-167 derived — are unchanged.

// Beacon-only cancel: the mask carries exactly the GuidedBullet bit (v83 shift
// 87 -> wire dword[1] 0x00800000) and nothing else.
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
		0x00, // nSecondaryStatChangedPoint
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("beacon-only cancel: got % x want % x", got, want)
	}
}

// A movement-affecting cancel (Speed) — the case that makes the trailing byte
// mandatory rather than optional.
func TestBuffCancelSpeedCarriesMovementByteV83(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 83, 1)
	cts := model.NewCharacterTemporaryStat()
	cts.AddStat(nil)(tn)(string(character.TemporaryStatTypeSpeed), 2001002, 20, 10, time.Now().Add(time.Minute))

	got := NewBuffCancel(*cts).Encode(nil, ctx)(nil)

	if len(got) != 17 {
		t.Fatalf("speed cancel length: got %d want 17 (mask + trailer)", len(got))
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

// v95 beacon-only cancel: exactly bit 127 (dword[0] 0x80000000).
func TestBuffCancelBeaconOnlyV95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	tn, _ := tenant.Create([16]byte{}, "GMS", 95, 1)
	cts := model.NewCharacterTemporaryStat()
	cts.AddStat(nil)(tn)(string(character.TemporaryStatTypeHomingBeacon), 5220011, 1000001, 10, time.Time{})

	got := NewBuffCancel(*cts).Encode(nil, ctx)(nil)

	want := append([]byte{0x00, 0x00, 0x00, 0x80}, make([]byte, 13)...)
	if !bytes.Equal(got, want) {
		t.Fatalf("v95 beacon-only cancel: got % x want % x", got, want)
	}
}
