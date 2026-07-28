package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/party"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v72 PARTY_OPERATION family verification — CWvsContext::OnPartyResult @0x934f3c,
// switch(Decode1(mode)) @0x934f9c (GMS_v72.1_U_DEVM.exe, port 13339).
//
// The v72 OnPartyResult mode table is NOT byte-identical to v79/v83. Every case
// byte below was read directly from the v72 switch @0x934f9c; each notice arm's
// identity was confirmed by decrypting its StringPool entry in-process
// (StringPool::GetString @0x6e26ea; key @0x9D5F54; whole-16-byte ring rotation
// per sub_6E294A @0x6e294a) and matching the plaintext to the arm's message.
//
// LOW arms (cases <= 0x11) are byte-identical to v83/v79:
//	AlreadyJoined1=9(SP320 "Already have joined a party.")  [done in error_test.go]
//	BeginnerCannotCreate=10(SP321 "A beginner can't create a party.")
//	NotInParty=13(SP322 "You have yet to join a party.")
//	AlreadyJoined2=16(SP320 "Already have joined a party.")
//	PartyFull=17(SP323 "The party you're trying to join is already in full capacity.")
//	UnableToFindInChannel=19(SP332 "Unable to find the requested character in this channel.") — mode-only, no DecodeStr.
//
// UPPER arms SHIFT -1 vs v79/v83 (v72 lacks the v83 CannotKick case-25 slot, so
// every case above it is one lower):
//	OnlyWithinVicinity=27(SP4024 "This can only be given to a party member within the vicinity.")   [v79 28]
//	UnableToHandOver=28(SP4026 "Unable to hand over the leadership post; ...")                        [v79 29]
//	OnlySameChannel=29(SP4025 "You may only change with the party member that's on the same channel.") [v79 30]
//	GmCannotCreate=31(SP328 "As a GM, you're forbidden from creating a party.")                        [v79 32]
//	UnableToFindCharacter=32(SP368 "Unable to find the character.")                                    [v79 33]
//
// CannotKick ("Cannot kick another user in this map", v83 case 25 / SP_5059) is
// VERSION-ABSENT in v72: every case @0x934f9c was enumerated and NONE decrypts to
// that text (the v72 upper block runs 0x1A/0x1B/0x1C/0x1D/0x1F/0x20 with no kick
// arm). It therefore gets no v72 fixture/marker/evidence.
//
// name arms (case 0x15/0x16/0x17 → DecodeStr(target) then StringPool notice; wire
// = [mode, name]): BlockingInvitations=21(SP309), TakingCareOfInvitation=22(SP2703),
// RequestDenied=23(SP310). Same modes as v79/v83.
//
// data arms (PARTYDATA / member-list bodies). PARTYDATA::Decode @0x4d0898 reads
// 298 bytes (0x12A) — identical to v83; the Created case-8 body reads
// Decode4+Decode4+Decode4+Decode2+Decode2 (partyId + door town/target + x/y); the
// Invite case-4 body reads partyId+name+autoJoin only (jobId/level gated >=87, off
// for v72). The shared encoders carry NO major<87 version gate, so each v72 encode
// is asserted byte-equal to the IDA-verified v83 encode (cross-version equality).

// packet-audit:verify packet=party/clientbound/PartyBeginnerCannotCreate version=gms_v72 ida=0x934f3c
// packet-audit:verify packet=party/clientbound/PartyNotInParty version=gms_v72 ida=0x934f3c
// packet-audit:verify packet=party/clientbound/PartyAlreadyJoined2 version=gms_v72 ida=0x934f3c
// packet-audit:verify packet=party/clientbound/PartyPartyFull version=gms_v72 ida=0x934f3c
// packet-audit:verify packet=party/clientbound/PartyUnableToFindInChannel version=gms_v72 ida=0x934f3c
// packet-audit:verify packet=party/clientbound/PartyOnlyWithinVicinity version=gms_v72 ida=0x934f3c
// packet-audit:verify packet=party/clientbound/PartyUnableToHandOver version=gms_v72 ida=0x934f3c
// packet-audit:verify packet=party/clientbound/PartyOnlySameChannel version=gms_v72 ida=0x934f3c
// packet-audit:verify packet=party/clientbound/PartyGmCannotCreate version=gms_v72 ida=0x934f3c
// packet-audit:verify packet=party/clientbound/PartyUnableToFindCharacter version=gms_v72 ida=0x934f3c
func TestPartyModeOnlyArmsV72(t *testing.T) {
	cases := map[byte][]byte{
		10: NewBeginnerCannotCreate(10).Encode(nil, nil)(nil),
		13: NewNotInParty(13).Encode(nil, nil)(nil),
		16: NewAlreadyJoined2(16).Encode(nil, nil)(nil),
		17: NewPartyFull(17).Encode(nil, nil)(nil),
		19: NewUnableToFindInChannel(19).Encode(nil, nil)(nil),
		27: NewOnlyWithinVicinity(27).Encode(nil, nil)(nil),
		28: NewUnableToHandOver(28).Encode(nil, nil)(nil),
		29: NewOnlySameChannel(29).Encode(nil, nil)(nil),
		31: NewGmCannotCreate(31).Encode(nil, nil)(nil),
		32: NewUnableToFindCharacter(32).Encode(nil, nil)(nil),
	}
	for mode, got := range cases {
		if !bytes.Equal(got, []byte{mode}) {
			t.Errorf("v72 party mode-only mode %d: got % x want %02x", mode, got, mode)
		}
	}
}

// packet-audit:verify packet=party/clientbound/PartyBlockingInvitations version=gms_v72 ida=0x934f3c
// packet-audit:verify packet=party/clientbound/PartyTakingCareOfInvitation version=gms_v72 ida=0x934f3c
// packet-audit:verify packet=party/clientbound/PartyRequestDenied version=gms_v72 ida=0x934f3c
func TestPartyNameArmsV72(t *testing.T) {
	// WriteByte(mode) + WriteAsciiString("Bob") = [mode, 03 00, 'B','o','b'].
	bob := []byte{0x03, 0x00, 0x42, 0x6F, 0x62}
	want := func(mode byte) []byte { return append([]byte{mode}, bob...) }
	if got := NewBlockingInvitations(21, "Bob").Encode(nil, nil)(nil); !bytes.Equal(got, want(21)) {
		t.Errorf("v72 BlockingInvitations: got % x want % x", got, want(21))
	}
	if got := NewTakingCareOfInvitation(22, "Bob").Encode(nil, nil)(nil); !bytes.Equal(got, want(22)) {
		t.Errorf("v72 TakingCareOfInvitation: got % x want % x", got, want(22))
	}
	if got := NewRequestDenied(23, "Bob").Encode(nil, nil)(nil); !bytes.Equal(got, want(23)) {
		t.Errorf("v72 RequestDenied: got % x want % x", got, want(23))
	}
}

// packet-audit:verify packet=party/clientbound/PartyCreated version=gms_v72 ida=0x934f3c
// packet-audit:verify packet=party/clientbound/PartyChangeLeader version=gms_v72 ida=0x934f3c
// packet-audit:verify packet=party/clientbound/PartyDisband version=gms_v72 ida=0x934f3c
// packet-audit:verify packet=party/clientbound/PartyInvite version=gms_v72 ida=0x934f3c
// packet-audit:verify packet=party/clientbound/PartyJoin version=gms_v72 ida=0x934f3c
// packet-audit:verify packet=party/clientbound/PartyLeft version=gms_v72 ida=0x934f3c
// packet-audit:verify packet=party/clientbound/PartyUpdate version=gms_v72 ida=0x934f3c
func TestPartyDataArmsV72(t *testing.T) {
	v72 := pt.CreateContext("GMS", 72, 1)
	v83 := pt.CreateContext("GMS", 83, 1)
	members := []party.PartyMember{
		{Id: 100, Name: "Player1", JobId: 111, Level: 50, ChannelId: 1, MapId: 100000},
		{Id: 200, Name: "Player2", JobId: 222, Level: 70, ChannelId: -2, MapId: 200000},
		{Id: 300, Name: "Player3", JobId: 333, Level: 90, ChannelId: 3, MapId: 300000},
	}
	type arm struct {
		name     string
		v72, v83 []byte
	}
	// Modes are the real v72 switch case bytes: Created=8, ChangeLeader=26,
	// Disband=12, Invite=4, Join=15, Left=12, Update=7. The equality check is
	// mode-agnostic (same mode on both sides); the real value is documentary.
	arms := []arm{
		{"Created", NewCreated(8, 12345).Encode(nil, v72)(nil), NewCreated(8, 12345).Encode(nil, v83)(nil)},
		{"ChangeLeader", NewChangeLeader(26, 9999, true).Encode(nil, v72)(nil), NewChangeLeader(26, 9999, true).Encode(nil, v83)(nil)},
		{"Disband", NewDisband(12, 5000, 300).Encode(nil, v72)(nil), NewDisband(12, 5000, 300).Encode(nil, v83)(nil)},
		{"Invite", NewInvite(4, 5000, "PartyLeader", 100, 50).Encode(nil, v72)(nil), NewInvite(4, 5000, "PartyLeader", 100, 50).Encode(nil, v83)(nil)},
		{"Join", NewJoin(15, 5000, "Player2", members, 100).Encode(nil, v72)(nil), NewJoin(15, 5000, "Player2", members, 100).Encode(nil, v83)(nil)},
		{"Left", NewLeft(12, 5000, 100, "Player1", false, members, 200).Encode(nil, v72)(nil), NewLeft(12, 5000, 100, "Player1", false, members, 200).Encode(nil, v83)(nil)},
		{"Update", NewUpdate(7, 5000, members, 100).Encode(nil, v72)(nil), NewUpdate(7, 5000, members, 100).Encode(nil, v83)(nil)},
	}
	for _, a := range arms {
		if !bytes.Equal(a.v72, a.v83) {
			t.Errorf("%s v72 != v83\n v72: % x\n v83: % x", a.name, a.v72, a.v83)
		}
	}
}
