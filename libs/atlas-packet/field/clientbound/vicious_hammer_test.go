package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Arm bodies IDA-verified in v83 (CUIItemUpgrade::OnPacket sub_82B2C3, reached
// via CField::OnItemUpgrade 0x537f8c), v87 (sub_88F348 via sub_88F332
// a2==375, reached via CField::OnItemUpgrade 0x55fa12 -> this[135] vtable
// slot 15), and v95 (CUIItemUpgrade::OnItemUpgradeResult 0x7c0fd0, reached
// via CUIItemUpgrade::OnPacket 0x7c2e10 when nType==425). Wire shapes are
// version-invariant; the mode byte is config-resolved in production (literal
// bytes below are test-only).
//
// Open  — mode(1) + token(4) + hammerCount(4) = 9 bytes
// Success — mode(1)=61 + flag(4) = 5 bytes
// Failure — mode(1)=62 + errorCode(4) = 5 bytes
//
// NOTE (task-129): the mode-byte LITERAL differs by version — v83 uses 61
// (success) / 62 (failure); v87 uses 63 (success) / 64 (failure), per live
// decompile of sub_88F348 (mode==63 branch, mode==64 switch); v95 uses 65
// (success) / 66 (failure), per live decompile of CUIItemUpgrade::
// OnItemUpgradeResult 0x7c0fd0. The byte SHAPE (mode + flag / mode +
// errorCode / mode + token + count) is version-invariant and confirmed
// identical across all three binaries — that is what these fixtures test
// (the mode field is a generic byte param, never a literal in production
// code). docs/packets/dispatchers/vicious_hammer.yaml's gms_v87 SUCCESS=63 /
// FAILURE=64 and gms_v95 SUCCESS=65 / FAILURE=66 modes match this decompile
// (corrected in task-129; the prior placeholder 61/62 entries for both
// versions were disproven).
//
// NOTE (task-129 gms_v84): v84 uses SUCCESS=61 / FAILURE=62 (same literals as
// v83), verified live in CUIItemUpgrade::OnItemUpgradeResult sub_85676C
// (mode==61 branch / mode==62 switch), reached via CUIItemUpgrade::OnPacket
// sub_856756 (header==364/0x16C) from the forwarder at 0x5443af (this[135];
// IDB symbol CField::OnCharacterSale, but functionally the hammer's
// CField::OnItemUpgrade in v84). The clientbound header is 364/0x16C — the
// prior template value 0x169 (361) was a DEAD opcode (it routes to the
// name/world-transfer dialog whose gate only handles 359/360); corrected to
// 0x16C in task-129.
//
// NOTE (task-129 gms_v79 extension): v79 SUCCESS=60 / FAILURE=61 IDA-verified
// live (port 13340) in CUIItemUpgrade::OnItemUpgradeResult 0x799d61
// (Decode1(mode); mode==60 success branch reads Decode4(flag), mode==61 failure
// branch reads Decode4(errorCode) switch 1=str5014 / 2=str5015 / 3=str5017 /
// default str5343, else OPEN reads Decode4(token)+Decode4(hammerCount)), reached
// via CUIItemUpgrade::OnPacket 0x799d4b (a2==330 / 0x14A). The byte SHAPE is
// version-invariant (the fixtures' literal 61/62 mode bytes are test-only — the
// mode field is a generic byte param, config-resolved in production); v79's
// SUCCESS=60 / FAILURE=61 literals live in docs/packets/dispatchers/
// vicious_hammer.yaml. Serverbound sender CUIItemUpgrade::Update 0x7998da
// COutPacket(250/0xFA).
// packet-audit:verify packet=field/clientbound/FieldViciousHammerOpen version=gms_v79 ida=0x799d61
// packet-audit:verify packet=field/clientbound/FieldViciousHammerSuccess version=gms_v79 ida=0x799d61
// packet-audit:verify packet=field/clientbound/FieldViciousHammerFailure version=gms_v79 ida=0x799d61
// packet-audit:verify packet=field/clientbound/FieldViciousHammerOpen version=gms_v83 ida=0x537f8c
// packet-audit:verify packet=field/clientbound/FieldViciousHammerSuccess version=gms_v83 ida=0x537f8c
// packet-audit:verify packet=field/clientbound/FieldViciousHammerFailure version=gms_v83 ida=0x537f8c
// packet-audit:verify packet=field/clientbound/FieldViciousHammerOpen version=gms_v84 ida=0x5443af
// packet-audit:verify packet=field/clientbound/FieldViciousHammerSuccess version=gms_v84 ida=0x5443af
// packet-audit:verify packet=field/clientbound/FieldViciousHammerFailure version=gms_v84 ida=0x5443af
// packet-audit:verify packet=field/clientbound/FieldViciousHammerOpen version=gms_v87 ida=0x55fa12
// packet-audit:verify packet=field/clientbound/FieldViciousHammerSuccess version=gms_v87 ida=0x55fa12
// packet-audit:verify packet=field/clientbound/FieldViciousHammerFailure version=gms_v87 ida=0x55fa12
// packet-audit:verify packet=field/clientbound/FieldViciousHammerOpen version=gms_v95 ida=0x52a430
// packet-audit:verify packet=field/clientbound/FieldViciousHammerSuccess version=gms_v95 ida=0x52a430
// packet-audit:verify packet=field/clientbound/FieldViciousHammerFailure version=gms_v95 ida=0x52a430
func TestViciousHammerOpenByteOutput(t *testing.T) {
	// token packs hammerSlot=1 (high int16), equipSlot=-5/0xFFFB (low int16).
	input := NewViciousHammerOpen(0, 0x0001FFFB, 1)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			got := input.Encode(nil, ctx)(nil)
			want := []byte{0x00, 0xFB, 0xFF, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00}
			if len(got) != len(want) {
				t.Fatalf("byte count: got %d, want %d", len(got), len(want))
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("byte %d: got 0x%02X, want 0x%02X", i, got[i], want[i])
				}
			}
		})
	}
}

func TestViciousHammerSuccessByteOutput(t *testing.T) {
	input := NewViciousHammerSuccess(61, 0)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			got := input.Encode(nil, ctx)(nil)
			want := []byte{0x3D, 0x00, 0x00, 0x00, 0x00}
			if len(got) != len(want) {
				t.Fatalf("byte count: got %d, want %d", len(got), len(want))
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("byte %d: got 0x%02X, want 0x%02X", i, got[i], want[i])
				}
			}
		})
	}
}

func TestViciousHammerFailureByteOutput(t *testing.T) {
	input := NewViciousHammerFailure(62, 2)
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			got := input.Encode(nil, ctx)(nil)
			want := []byte{0x3E, 0x02, 0x00, 0x00, 0x00}
			if len(got) != len(want) {
				t.Fatalf("byte count: got %d, want %d", len(got), len(want))
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("byte %d: got 0x%02X, want 0x%02X", i, got[i], want[i])
				}
			}
		})
	}
}

func TestViciousHammerRoundTrips(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)

			open := NewViciousHammerOpen(0, 0x0001FFFB, 1)
			openOut := ViciousHammerOpen{}
			pt.RoundTrip(t, ctx, open.Encode, openOut.Decode, nil)
			if openOut.Token() != open.Token() || openOut.HammerCount() != open.HammerCount() {
				t.Errorf("open: got token %d count %d, want %d %d", openOut.Token(), openOut.HammerCount(), open.Token(), open.HammerCount())
			}

			success := NewViciousHammerSuccess(61, 0)
			successOut := ViciousHammerSuccess{}
			pt.RoundTrip(t, ctx, success.Encode, successOut.Decode, nil)
			if successOut.Flag() != success.Flag() {
				t.Errorf("success: got flag %d, want %d", successOut.Flag(), success.Flag())
			}

			failure := NewViciousHammerFailure(62, 3)
			failureOut := ViciousHammerFailure{}
			pt.RoundTrip(t, ctx, failure.Encode, failureOut.Decode, nil)
			if failureOut.ErrorCode() != failure.ErrorCode() {
				t.Errorf("failure: got code %d, want %d", failureOut.ErrorCode(), failure.ErrorCode())
			}
		})
	}
}
