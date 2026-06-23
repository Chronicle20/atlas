package clientbound

import (
	"bytes"
	"testing"
)

// ---------------------------------------------------------------------------
// Per-version byte fixtures for the discrete OnPartyResult error/notice arms.
//
// Mode bytes are taken from docs/packets/dispatchers/party.yaml (IDA-enumerated).
// v83/v84 are byte-identical; v87/v95/jms carry the non-uniform mode shift for
// the cases >= 16 (the +1 begins at ALREADY_HAVE_JOINED_A_PARTY_2). "—" in the
// spec table = version-absent: that version's fixture (and verify marker) is
// omitted. Read orders cited per struct in error.go. Dispatcher roots:
// v83 OnPartyResult@0xa3e31c; v84@0xa89cf3; v87@0xad697a; v95@0xa10ab0; jms@0xb297e7.

// modeOnlyFixture asserts a discrete mode-only struct encodes to exactly its mode byte.
func modeOnlyFixture(t *testing.T, mode byte, enc func(byte) []byte) {
	t.Helper()
	got := enc(mode)
	want := []byte{mode}
	if !bytes.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// modeOnlyArmCase couples a mode-only struct's per-version mode bytes to its
// encoder. A zero (absent) entry for a version means that version is skipped.
type modeOnlyArmCase struct {
	v83, v84, v87, v95, jms byte
	encode                  func(byte) []byte
}

func runModeOnly(t *testing.T, name string, c modeOnlyArmCase) {
	t.Helper()
	versions := []struct {
		label string
		mode  byte
	}{
		{"gms_v83", c.v83},
		{"gms_v84", c.v84},
		{"gms_v87", c.v87},
		{"gms_v95", c.v95},
		{"jms_v185", c.jms},
	}
	for _, ver := range versions {
		if ver.mode == 0 { // version-absent ("—") — skip
			continue
		}
		ver := ver
		t.Run(name+"/"+ver.label, func(t *testing.T) { modeOnlyFixture(t, ver.mode, c.encode) })
	}
}

// TestModeOnlyPartyErrorArms covers the 12 mode-only OnPartyResult arms. Each
// arm's encoder writes exactly one mode byte; the per-version mode bytes are
// asserted from party.yaml.
//
// packet-audit:verify packet=party/clientbound/PartyAlreadyJoined1 version=gms_v83 ida=0xa3e31c
// packet-audit:verify packet=party/clientbound/PartyAlreadyJoined1 version=gms_v84 ida=0xa89cf3
// packet-audit:verify packet=party/clientbound/PartyAlreadyJoined1 version=gms_v87 ida=0xad697a
// packet-audit:verify packet=party/clientbound/PartyAlreadyJoined1 version=gms_v95 ida=0xa10ab0
// packet-audit:verify packet=party/clientbound/PartyAlreadyJoined1 version=jms_v185 ida=0xb297e7
// packet-audit:verify packet=party/clientbound/PartyBeginnerCannotCreate version=gms_v83 ida=0xa3e31c
// packet-audit:verify packet=party/clientbound/PartyBeginnerCannotCreate version=gms_v84 ida=0xa89cf3
// packet-audit:verify packet=party/clientbound/PartyBeginnerCannotCreate version=gms_v87 ida=0xad697a
// packet-audit:verify packet=party/clientbound/PartyBeginnerCannotCreate version=gms_v95 ida=0xa10ab0
// packet-audit:verify packet=party/clientbound/PartyBeginnerCannotCreate version=jms_v185 ida=0xb297e7
// packet-audit:verify packet=party/clientbound/PartyNotInParty version=gms_v83 ida=0xa3e31c
// packet-audit:verify packet=party/clientbound/PartyNotInParty version=gms_v84 ida=0xa89cf3
// packet-audit:verify packet=party/clientbound/PartyNotInParty version=gms_v87 ida=0xad697a
// packet-audit:verify packet=party/clientbound/PartyNotInParty version=gms_v95 ida=0xa10ab0
// packet-audit:verify packet=party/clientbound/PartyNotInParty version=jms_v185 ida=0xb297e7
// packet-audit:verify packet=party/clientbound/PartyAlreadyJoined2 version=gms_v83 ida=0xa3e31c
// packet-audit:verify packet=party/clientbound/PartyAlreadyJoined2 version=gms_v84 ida=0xa89cf3
// packet-audit:verify packet=party/clientbound/PartyAlreadyJoined2 version=gms_v87 ida=0xad697a
// packet-audit:verify packet=party/clientbound/PartyAlreadyJoined2 version=gms_v95 ida=0xa10ab0
// packet-audit:verify packet=party/clientbound/PartyAlreadyJoined2 version=jms_v185 ida=0xb297e7
// packet-audit:verify packet=party/clientbound/PartyPartyFull version=gms_v83 ida=0xa3e31c
// packet-audit:verify packet=party/clientbound/PartyPartyFull version=gms_v84 ida=0xa89cf3
// packet-audit:verify packet=party/clientbound/PartyPartyFull version=gms_v87 ida=0xad697a
// packet-audit:verify packet=party/clientbound/PartyPartyFull version=gms_v95 ida=0xa10ab0
// packet-audit:verify packet=party/clientbound/PartyPartyFull version=jms_v185 ida=0xb297e7
// packet-audit:verify packet=party/clientbound/PartyUnableToFindInChannel version=gms_v83 ida=0xa3e31c
// packet-audit:verify packet=party/clientbound/PartyUnableToFindInChannel version=gms_v84 ida=0xa89cf3
// packet-audit:verify packet=party/clientbound/PartyCannotKick version=gms_v83 ida=0xa3e31c
// packet-audit:verify packet=party/clientbound/PartyCannotKick version=gms_v84 ida=0xa89cf3
// packet-audit:verify packet=party/clientbound/PartyCannotKick version=gms_v87 ida=0xad697a
// packet-audit:verify packet=party/clientbound/PartyCannotKick version=gms_v95 ida=0xa10ab0
// packet-audit:verify packet=party/clientbound/PartyCannotKick version=jms_v185 ida=0xb297e7
// packet-audit:verify packet=party/clientbound/PartyOnlyWithinVicinity version=gms_v83 ida=0xa3e31c
// packet-audit:verify packet=party/clientbound/PartyOnlyWithinVicinity version=gms_v84 ida=0xa89cf3
// packet-audit:verify packet=party/clientbound/PartyOnlyWithinVicinity version=gms_v87 ida=0xad697a
// packet-audit:verify packet=party/clientbound/PartyOnlyWithinVicinity version=gms_v95 ida=0xa10ab0
// packet-audit:verify packet=party/clientbound/PartyOnlyWithinVicinity version=jms_v185 ida=0xb297e7
// packet-audit:verify packet=party/clientbound/PartyUnableToHandOver version=gms_v83 ida=0xa3e31c
// packet-audit:verify packet=party/clientbound/PartyUnableToHandOver version=gms_v84 ida=0xa89cf3
// packet-audit:verify packet=party/clientbound/PartyUnableToHandOver version=gms_v87 ida=0xad697a
// packet-audit:verify packet=party/clientbound/PartyUnableToHandOver version=gms_v95 ida=0xa10ab0
// packet-audit:verify packet=party/clientbound/PartyUnableToHandOver version=jms_v185 ida=0xb297e7
// packet-audit:verify packet=party/clientbound/PartyOnlySameChannel version=gms_v83 ida=0xa3e31c
// packet-audit:verify packet=party/clientbound/PartyOnlySameChannel version=gms_v84 ida=0xa89cf3
// packet-audit:verify packet=party/clientbound/PartyOnlySameChannel version=gms_v87 ida=0xad697a
// packet-audit:verify packet=party/clientbound/PartyOnlySameChannel version=gms_v95 ida=0xa10ab0
// packet-audit:verify packet=party/clientbound/PartyOnlySameChannel version=jms_v185 ida=0xb297e7
// packet-audit:verify packet=party/clientbound/PartyGmCannotCreate version=gms_v83 ida=0xa3e31c
// packet-audit:verify packet=party/clientbound/PartyGmCannotCreate version=gms_v84 ida=0xa89cf3
// packet-audit:verify packet=party/clientbound/PartyGmCannotCreate version=gms_v87 ida=0xad697a
// packet-audit:verify packet=party/clientbound/PartyGmCannotCreate version=gms_v95 ida=0xa10ab0
// packet-audit:verify packet=party/clientbound/PartyGmCannotCreate version=jms_v185 ida=0xb297e7
// packet-audit:verify packet=party/clientbound/PartyUnableToFindCharacter version=gms_v83 ida=0xa3e31c
// packet-audit:verify packet=party/clientbound/PartyUnableToFindCharacter version=gms_v84 ida=0xa89cf3
// packet-audit:verify packet=party/clientbound/PartyUnableToFindCharacter version=gms_v87 ida=0xad697a
// packet-audit:verify packet=party/clientbound/PartyUnableToFindCharacter version=gms_v95 ida=0xa10ab0
func TestModeOnlyPartyErrorArms(t *testing.T) {
	cases := map[string]modeOnlyArmCase{
		"AlreadyJoined1":        {9, 9, 9, 9, 9, func(b byte) []byte { m := NewAlreadyJoined1(b); return m.Encode(nil, nil)(nil) }},
		"BeginnerCannotCreate":  {10, 10, 10, 10, 10, func(b byte) []byte { m := NewBeginnerCannotCreate(b); return m.Encode(nil, nil)(nil) }},
		"NotInParty":            {13, 13, 13, 13, 13, func(b byte) []byte { m := NewNotInParty(b); return m.Encode(nil, nil)(nil) }},
		"AlreadyJoined2":        {16, 16, 17, 17, 17, func(b byte) []byte { m := NewAlreadyJoined2(b); return m.Encode(nil, nil)(nil) }},
		"PartyFull":             {17, 17, 18, 18, 18, func(b byte) []byte { m := NewPartyFull(b); return m.Encode(nil, nil)(nil) }},
		"UnableToFindInChannel": {19, 19, 0, 0, 0, func(b byte) []byte { m := NewUnableToFindInChannel(b); return m.Encode(nil, nil)(nil) }},
		"CannotKick":            {25, 25, 29, 29, 29, func(b byte) []byte { m := NewCannotKick(b); return m.Encode(nil, nil)(nil) }},
		"OnlyWithinVicinity":    {28, 28, 32, 32, 32, func(b byte) []byte { m := NewOnlyWithinVicinity(b); return m.Encode(nil, nil)(nil) }},
		"UnableToHandOver":      {29, 29, 33, 33, 33, func(b byte) []byte { m := NewUnableToHandOver(b); return m.Encode(nil, nil)(nil) }},
		"OnlySameChannel":       {30, 30, 34, 34, 34, func(b byte) []byte { m := NewOnlySameChannel(b); return m.Encode(nil, nil)(nil) }},
		"GmCannotCreate":        {32, 32, 36, 36, 36, func(b byte) []byte { m := NewGmCannotCreate(b); return m.Encode(nil, nil)(nil) }},
		"UnableToFindCharacter": {33, 33, 37, 37, 0, func(b byte) []byte { m := NewUnableToFindCharacter(b); return m.Encode(nil, nil)(nil) }},
	}
	for name, c := range cases {
		runModeOnly(t, name, c)
	}
}

// --- Invite-target arms ({mode,name}) -----------------------------------------

// TestInviteTargetPartyArms covers the 3 invite-target arms (v83/v84 only). Each
// writes mode + WriteAsciiString(name): a 2-byte little-endian length prefix
// followed by the ShiftJIS-encoded ascii bytes. "Bob" → [0x03, 0x00, 'B','o','b'].
//
// packet-audit:verify packet=party/clientbound/PartyBlockingInvitations version=gms_v83 ida=0xa3e31c
// packet-audit:verify packet=party/clientbound/PartyBlockingInvitations version=gms_v84 ida=0xa89cf3
// packet-audit:verify packet=party/clientbound/PartyTakingCareOfInvitation version=gms_v83 ida=0xa3e31c
// packet-audit:verify packet=party/clientbound/PartyTakingCareOfInvitation version=gms_v84 ida=0xa89cf3
// packet-audit:verify packet=party/clientbound/PartyRequestDenied version=gms_v83 ida=0xa3e31c
// packet-audit:verify packet=party/clientbound/PartyRequestDenied version=gms_v84 ida=0xa89cf3
func TestInviteTargetPartyArms(t *testing.T) {
	// mode byte + 2-byte ascii length prefix + "Bob" (3 bytes) = 6 bytes total.
	want := func(mode byte) []byte { return []byte{mode, 0x03, 0x00, 'B', 'o', 'b'} }
	cases := []struct {
		name     string
		v83, v84 byte
		encode   func(byte) []byte
	}{
		{"BlockingInvitations", 21, 21, func(b byte) []byte {
			m := NewBlockingInvitations(b, "Bob")
			return m.Encode(nil, nil)(nil)
		}},
		{"TakingCareOfInvitation", 22, 22, func(b byte) []byte {
			m := NewTakingCareOfInvitation(b, "Bob")
			return m.Encode(nil, nil)(nil)
		}},
		{"RequestDenied", 23, 23, func(b byte) []byte {
			m := NewRequestDenied(b, "Bob")
			return m.Encode(nil, nil)(nil)
		}},
	}
	for _, c := range cases {
		t.Run(c.name+"/gms_v83", func(t *testing.T) {
			if got := c.encode(c.v83); !bytes.Equal(got, want(c.v83)) {
				t.Fatalf("got %v want %v", got, want(c.v83))
			}
		})
		t.Run(c.name+"/gms_v84", func(t *testing.T) {
			if got := c.encode(c.v84); !bytes.Equal(got, want(c.v84)) {
				t.Fatalf("got %v want %v", got, want(c.v84))
			}
		})
	}
}

// TestPartyErrorDivergence documents the D8 (IDA wins) migration from the old
// shared Error{mode,name} catch-all to the discrete structs.
//
//   - Name-bearing arms (cases 21/22/23) stay byte-preserving: the old Error
//     wrote WriteByte(mode)+WriteAsciiString(name); RequestDenied writes the
//     same. The expected bytes below are exactly what NewError(mode,name) used
//     to produce (inlined because NewError is deleted in this task).
//   - The two "unable to find" arms become MODE-ONLY: the v83 switch (case 33
//     and case 19) reads no trailing DecodeStr, so the trailing name the legacy
//     Error wrote is intentionally dropped. This is the only intended byte
//     divergence, scoped to bytes the client never consumed.
func TestPartyErrorDivergence(t *testing.T) {
	// Byte-preserving: RequestDenied == legacy Error{mode,name}.
	// Legacy Error.Encode wrote: WriteByte(mode) + WriteAsciiString(name).
	// For mode=23, name="Bob": [23, 0x03,0x00, 'B','o','b'].
	const mode byte = 23
	legacyErrorBytes := []byte{mode, 0x03, 0x00, 'B', 'o', 'b'}
	if got := NewRequestDenied(mode, "Bob").Encode(nil, nil)(nil); !bytes.Equal(got, legacyErrorBytes) {
		t.Fatalf("RequestDenied not byte-identical to legacy Error: got %v want %v", got, legacyErrorBytes)
	}

	// D8 divergence: mode-only arms write ONLY the mode byte. The legacy Error
	// wrote a trailing name here (e.g. [33, len, ...ascii]); that name is gone.
	if got := NewUnableToFindCharacter(33).Encode(nil, nil)(nil); !bytes.Equal(got, []byte{33}) {
		t.Fatalf("UnableToFindCharacter must be mode-only: got %v want [33]", got)
	}
	if got := NewUnableToFindInChannel(19).Encode(nil, nil)(nil); !bytes.Equal(got, []byte{19}) {
		t.Fatalf("UnableToFindInChannel must be mode-only: got %v want [19]", got)
	}
}
