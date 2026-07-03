package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Arm bodies IDA-verified in v83 (CUIItemUpgrade::OnPacket sub_82B2C3, reached
// via CField::OnItemUpgrade 0x537f8c) and v95 (CUIItemUpgrade::ShowResult
// 0x7bec20). Wire shapes are version-invariant; the mode byte is
// config-resolved in production (literal bytes below are test-only).
//
// Open  — mode(1) + token(4) + hammerCount(4) = 9 bytes
// Success — mode(1)=61 + flag(4) = 5 bytes
// Failure — mode(1)=62 + errorCode(4) = 5 bytes
//
// NOTE (task-129): the mode-byte LITERAL differs by version — v83 uses 61
// (success) / 62 (failure); v95 uses 65 (success) / 66 (failure), per live
// decompile of CUIItemUpgrade::OnItemUpgradeResult 0x7c0fd0. The byte SHAPE
// (mode + flag / mode + errorCode / mode + token + count) is version-invariant
// and confirmed identical in both binaries — that is what these fixtures test
// (the mode field is a generic byte param, never a literal in production
// code). See task-129 report for the full contradiction against
// docs/packets/dispatchers/vicious_hammer.yaml's gms_v95 SUCCESS/FAILURE
// modes (currently 61/62 — unverified/incorrect for v95).
// packet-audit:verify packet=field/clientbound/FieldViciousHammerOpen version=gms_v83 ida=0x537f8c
// packet-audit:verify packet=field/clientbound/FieldViciousHammerSuccess version=gms_v83 ida=0x537f8c
// packet-audit:verify packet=field/clientbound/FieldViciousHammerFailure version=gms_v83 ida=0x537f8c
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
