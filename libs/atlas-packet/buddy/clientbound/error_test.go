package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// ---------------------------------------------------------------------------
// Per-version byte fixtures for the discrete OnFriendResult error/notice arms.
//
// Mode bytes are taken from docs/packets/dispatchers/buddy.yaml (IDA-enumerated)
// and are BYTE-IDENTICAL across all 5 versions (the buddy mode table is NOT
// shifted in v95, unlike party/guild). Per-version OnFriendResult addrs:
// gms_v83 0xa3f2e8, gms_v84 0xa8ada2, gms_v87 0xad7ae5, gms_v95 0xa12630,
// jms_v185 0xb2a873.
//
// The 5 mode-only arms (ListFull/OtherListFull/AlreadyBuddy/CannotBuddyGm/
// CharacterNotFound) write only the mode byte in every version. The 4
// extra-byte arms (UnknownError{,2,3,4}) write a trailing 0 in GMS (the arm
// reads CInPacket::Decode1) but are MODE-ONLY in JMS (buddy.yaml).

// --- Mode-only arms -----------------------------------------------------------

// TestModeOnlyBuddyErrorArms covers the 5 mode-only OnFriendResult error arms.
// Each encodes to exactly its mode byte for every version.
//
// packet-audit:verify packet=buddy/clientbound/BuddyListFull version=gms_v83 ida=0xa3f2e8
// packet-audit:verify packet=buddy/clientbound/BuddyListFull version=gms_v84 ida=0xa8ada2
// packet-audit:verify packet=buddy/clientbound/BuddyListFull version=gms_v87 ida=0xad7ae5
// packet-audit:verify packet=buddy/clientbound/BuddyListFull version=gms_v95 ida=0xa12630
// packet-audit:verify packet=buddy/clientbound/BuddyListFull version=jms_v185 ida=0xb2a873
// packet-audit:verify packet=buddy/clientbound/BuddyOtherListFull version=gms_v83 ida=0xa3f2e8
// packet-audit:verify packet=buddy/clientbound/BuddyOtherListFull version=gms_v84 ida=0xa8ada2
// packet-audit:verify packet=buddy/clientbound/BuddyOtherListFull version=gms_v87 ida=0xad7ae5
// packet-audit:verify packet=buddy/clientbound/BuddyOtherListFull version=gms_v95 ida=0xa12630
// packet-audit:verify packet=buddy/clientbound/BuddyOtherListFull version=jms_v185 ida=0xb2a873
// packet-audit:verify packet=buddy/clientbound/BuddyAlreadyBuddy version=gms_v83 ida=0xa3f2e8
// packet-audit:verify packet=buddy/clientbound/BuddyAlreadyBuddy version=gms_v84 ida=0xa8ada2
// packet-audit:verify packet=buddy/clientbound/BuddyAlreadyBuddy version=gms_v87 ida=0xad7ae5
// packet-audit:verify packet=buddy/clientbound/BuddyAlreadyBuddy version=gms_v95 ida=0xa12630
// packet-audit:verify packet=buddy/clientbound/BuddyAlreadyBuddy version=jms_v185 ida=0xb2a873
// packet-audit:verify packet=buddy/clientbound/BuddyCannotBuddyGm version=gms_v83 ida=0xa3f2e8
// packet-audit:verify packet=buddy/clientbound/BuddyCannotBuddyGm version=gms_v84 ida=0xa8ada2
// packet-audit:verify packet=buddy/clientbound/BuddyCannotBuddyGm version=gms_v87 ida=0xad7ae5
// packet-audit:verify packet=buddy/clientbound/BuddyCannotBuddyGm version=gms_v95 ida=0xa12630
// packet-audit:verify packet=buddy/clientbound/BuddyCannotBuddyGm version=jms_v185 ida=0xb2a873
// packet-audit:verify packet=buddy/clientbound/BuddyCharacterNotFound version=gms_v83 ida=0xa3f2e8
// packet-audit:verify packet=buddy/clientbound/BuddyCharacterNotFound version=gms_v84 ida=0xa8ada2
// packet-audit:verify packet=buddy/clientbound/BuddyCharacterNotFound version=gms_v87 ida=0xad7ae5
// packet-audit:verify packet=buddy/clientbound/BuddyCharacterNotFound version=gms_v95 ida=0xa12630
// packet-audit:verify packet=buddy/clientbound/BuddyCharacterNotFound version=jms_v185 ida=0xb2a873
// TestBuddyAlreadyBuddyV79 pins the gms_v79 BUDDYLIST (op 60) ALREADY_BUDDY arm.
//
// IDA-verified (GMS_v79_1_DEVM.exe, port 13340) — CWvsContext::OnFriendResult
// @0x98854f: switch(Decode1(mode)) @0x98857a. case 0xDu @0x9888c6 calls only
// StringPool::GetInstance(722) + Notice and reads NOTHING further off the wire
// → mode-only. v79 case byte = 13 (0xD), same as v83. atlas AlreadyBuddy.Encode
// writes just WriteByte(mode).
//
// packet-audit:verify packet=buddy/clientbound/BuddyAlreadyBuddy version=gms_v79 ida=0x98854f
func TestBuddyAlreadyBuddyV79(t *testing.T) {
	const v79Mode = 13
	got := NewAlreadyBuddy(v79Mode).Encode(nil, nil)(nil)
	if want := []byte{v79Mode}; !bytes.Equal(got, want) {
		t.Fatalf("v79 BuddyAlreadyBuddy: got %v want %v", got, want)
	}
}

// TestBuddyAlreadyBuddyV72 pins the gms_v72 BUDDYLIST ALREADY_BUDDY arm.
//
// IDA-verified (GMS_v72.1_U_DEVM.exe, port 13339) — CWvsContext::OnFriendResult
// @0x935ecf: switch(Decode1(mode)) @0x935efa. case 0xDu @0x936246 sets up
// StringPool::GetInstance(722), then LABEL_39 @0x936284 calls GetString +
// CUtilDlg::Notice and reads NOTHING further off the wire → mode-only. v72 case
// byte = 13 (0xD), same as v79/v83. atlas AlreadyBuddy.Encode writes just
// WriteByte(mode).
//
// packet-audit:verify packet=buddy/clientbound/BuddyAlreadyBuddy version=gms_v72 ida=0x935ecf
func TestBuddyAlreadyBuddyV72(t *testing.T) {
	const v72Mode = 13
	got := NewAlreadyBuddy(v72Mode).Encode(nil, nil)(nil)
	if want := []byte{v72Mode}; !bytes.Equal(got, want) {
		t.Fatalf("v72 BuddyAlreadyBuddy: got %v want %v", got, want)
	}
}

func TestModeOnlyBuddyErrorArms(t *testing.T) {
	cases := map[string]struct {
		mode   byte
		encode func(byte) []byte
	}{
		"ListFull":          {11, func(b byte) []byte { return NewListFull(b).Encode(nil, nil)(nil) }},
		"OtherListFull":     {12, func(b byte) []byte { return NewOtherListFull(b).Encode(nil, nil)(nil) }},
		"AlreadyBuddy":      {13, func(b byte) []byte { return NewAlreadyBuddy(b).Encode(nil, nil)(nil) }},
		"CannotBuddyGm":     {14, func(b byte) []byte { return NewCannotBuddyGm(b).Encode(nil, nil)(nil) }},
		"CharacterNotFound": {15, func(b byte) []byte { return NewCharacterNotFound(b).Encode(nil, nil)(nil) }},
	}
	for name, c := range cases {
		for _, v := range pt.Variants {
			v := v
			c := c
			t.Run(name+"/"+v.Name, func(t *testing.T) {
				got := c.encode(c.mode)
				want := []byte{c.mode}
				if !bytes.Equal(got, want) {
					t.Fatalf("%s/%s: got %v want %v", name, v.Name, got, want)
				}
			})
		}
	}
}

// --- Extra-byte arms (version-gated) ------------------------------------------

// TestExtraByteBuddyErrorArms covers the 4 UNKNOWN_ERROR family arms. In GMS the
// arm reads a trailing CInPacket::Decode1, so the encoder writes [mode, 0x00];
// in JMS the arm is mode-only, so the encoder writes [mode].
//
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError version=gms_v83 ida=0xa3f2e8
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError version=gms_v84 ida=0xa8ada2
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError version=gms_v87 ida=0xad7ae5
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError version=gms_v95 ida=0xa12630
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError version=jms_v185 ida=0xb2a873
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError2 version=gms_v83 ida=0xa3f2e8
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError2 version=gms_v84 ida=0xa8ada2
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError2 version=gms_v87 ida=0xad7ae5
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError2 version=gms_v95 ida=0xa12630
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError2 version=jms_v185 ida=0xb2a873
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError3 version=gms_v83 ida=0xa3f2e8
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError3 version=gms_v84 ida=0xa8ada2
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError3 version=gms_v87 ida=0xad7ae5
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError3 version=gms_v95 ida=0xa12630
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError3 version=jms_v185 ida=0xb2a873
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError4 version=gms_v83 ida=0xa3f2e8
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError4 version=gms_v84 ida=0xa8ada2
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError4 version=gms_v87 ida=0xad7ae5
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError4 version=gms_v95 ida=0xa12630
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError4 version=jms_v185 ida=0xb2a873
func TestExtraByteBuddyErrorArms(t *testing.T) {
	cases := []struct {
		name   string
		mode   byte
		encode func(mode byte, region string, major, minor uint16) []byte
	}{
		{"UnknownError", 16, func(mode byte, region string, major, minor uint16) []byte {
			ctx := pt.CreateContext(region, major, minor)
			return NewUnknownError(mode).Encode(nil, ctx)(nil)
		}},
		{"UnknownError2", 17, func(mode byte, region string, major, minor uint16) []byte {
			ctx := pt.CreateContext(region, major, minor)
			return NewUnknownError2(mode).Encode(nil, ctx)(nil)
		}},
		{"UnknownError3", 19, func(mode byte, region string, major, minor uint16) []byte {
			ctx := pt.CreateContext(region, major, minor)
			return NewUnknownError3(mode).Encode(nil, ctx)(nil)
		}},
		{"UnknownError4", 22, func(mode byte, region string, major, minor uint16) []byte {
			ctx := pt.CreateContext(region, major, minor)
			return NewUnknownError4(mode).Encode(nil, ctx)(nil)
		}},
	}
	for _, c := range cases {
		for _, v := range pt.Variants {
			v := v
			c := c
			t.Run(c.name+"/"+v.Name, func(t *testing.T) {
				got := c.encode(c.mode, v.Region, v.MajorVersion, v.MinorVersion)
				var want []byte
				if v.Region == "GMS" {
					want = []byte{c.mode, 0x00} // GMS reads a trailing Decode1
				} else {
					want = []byte{c.mode} // JMS mode-only
				}
				if !bytes.Equal(got, want) {
					t.Fatalf("%s/%s: got %v want %v", c.name, v.Name, got, want)
				}
			})
		}
	}
}

// TestBuddyErrorRoundTrip exercises the version-gated Decode mirror for the
// extra-byte arms (GMS consumes the trailing byte; JMS does not). A clean
// RoundTrip (no unconsumed bytes) confirms Encode/Decode agree per region.
func TestBuddyErrorRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		v := v
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewUnknownError(16)
			output := UnknownError{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

// TestBuddyErrorDivergence documents the IDA-correct fix from the old shared
// Error{mode,hasExtra} catch-all to the discrete structs.
//
//   - Byte-preserving for the one live arm: UnknownError under GMS writes
//     [16, 0x00] — exactly what legacy NewBuddyError(16, true) produced in GMS.
//   - Latent correctness fix: the legacy hasExtra gate only set the extra byte
//     for UNKNOWN_ERROR (0x10). UnknownError2/3/4 now write the extra byte in
//     GMS too (buddy.yaml: cases 0x11/0x13/0x16 all read CInPacket::Decode1).
//     These three have no live caller (future-feature), so this is a latent
//     correctness fix, not a behavior change.
//   - JMS mode-only: legacy NewBuddyError(16, true) wrongly wrote [16, 0] on
//     jms; IDA (0xb2a873) says the jms arm is mode-only, so UnknownError writes
//     [16] under a jms ctx.
func TestBuddyErrorDivergence(t *testing.T) {
	gms := pt.CreateContext("GMS", 83, 1)
	jms := pt.CreateContext("JMS", 185, 1)

	// Legacy NewBuddyError(16, true) under GMS produced WriteByte(16)+WriteByte(0).
	legacyUnknownErrorGMS := []byte{16, 0x00}
	if got := NewUnknownError(16).Encode(nil, gms)(nil); !bytes.Equal(got, legacyUnknownErrorGMS) {
		t.Fatalf("UnknownError GMS not byte-identical to legacy NewBuddyError(16,true): got %v want %v", got, legacyUnknownErrorGMS)
	}

	// Latent fix: UnknownError2/3/4 now also write the extra byte in GMS.
	if got := NewUnknownError2(17).Encode(nil, gms)(nil); !bytes.Equal(got, []byte{17, 0x00}) {
		t.Fatalf("UnknownError2 GMS: got %v want [17 0]", got)
	}
	if got := NewUnknownError3(19).Encode(nil, gms)(nil); !bytes.Equal(got, []byte{19, 0x00}) {
		t.Fatalf("UnknownError3 GMS: got %v want [19 0]", got)
	}
	if got := NewUnknownError4(22).Encode(nil, gms)(nil); !bytes.Equal(got, []byte{22, 0x00}) {
		t.Fatalf("UnknownError4 GMS: got %v want [22 0]", got)
	}

	// JMS mode-only: legacy wrongly wrote [16,0] here; IDA says [16].
	if got := NewUnknownError(16).Encode(nil, jms)(nil); !bytes.Equal(got, []byte{16}) {
		t.Fatalf("UnknownError JMS must be mode-only: got %v want [16]", got)
	}
}
