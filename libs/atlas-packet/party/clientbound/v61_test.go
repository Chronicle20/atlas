package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/party"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v61 PARTY_OPERATION (clientbound op 59) family verification —
// CWvsContext::OnPacket @0x8303eb case 59 → CWvsContext::OnPartyResult @0x857a8c,
// switch(Decode1(mode)) @0x857aec (GMS_v61.1_U_DEVM.exe, port 13338).
//
// The v61 OnPartyResult mode table is BYTE-IDENTICAL to the IDA-verified v72
// switch: every case byte and body shape below was read directly from the v61
// decompile @0x857a8c. Only the StringPool notice ids are renumbered (the low
// 300-series notices shift +13 vs v72: SP320→333, SP321→334, SP322→335,
// SP323→336, SP332→345, SP328→341) — a table renumbering that never touches the
// wire. The mode bytes are unshifted from v72, so the atlas struct↔mode mapping
// mirrors the v72 switch exactly (case-for-case, same body shape).
//
//	mode-only notice arms (case → StringPool id, NO further wire read):
//	  AlreadyJoined1=9 (SP333 @0x858691), BeginnerCannotCreate=10 (SP334 @0x8586be),
//	  NotInParty=13 (SP335 @0x8586e8), AlreadyJoined2=16 (SP333 @0x858712),
//	  PartyFull=17 (SP336 @0x85873c), UnableToFindInChannel=19 (SP345 @0x8587f9),
//	  OnlyWithinVicinity=27 (SP3971 @0x85860a), UnableToHandOver=28 (SP3973 @0x858637),
//	  OnlySameChannel=29 (SP3972 @0x858664), GmCannotCreate=31 (SP341 @0x8587a5),
//	  UnableToFindCharacter=32 (SP382 @0x858829). Each case body is
//	  sub_678022(&s, id)+sub_47010A (chat-log notice) and reads NOTHING further
//	  off the wire → mode-only.
//	name arms (case 0x15/0x16/0x17 → DecodeStr(target) then StringPool notice;
//	  wire = [mode, name]): BlockingInvitations=21 (DecodeStr @0x857cb1, SP322),
//	  TakingCareOfInvitation=22 (DecodeStr @0x857d1d, SP2662),
//	  RequestDenied=23 (DecodeStr @0x857d83, SP323).
//	data arms (PARTYDATA / member-list bodies, read order in the case bodies):
//	  Created=8 (@0x857df2: Decode4 partyId + Decode4 + Decode4 + Decode2 + Decode2),
//	  ChangeLeader=26 (@0x858454: Decode4 id + Decode1 bool),
//	  Disband/Left=12 (@0x857f67: Decode4 partyId + Decode4 + Decode1 + PARTYDATA::Decode),
//	  Invite=4 (@0x857afa: Decode4 partyId + DecodeStr name + Decode1 autoJoin; jobId/level
//	    gated >=87, off for v61 — invite.go v87plus),
//	  Join=15 (@0x85822f: Decode4 partyId + DecodeStr + PARTYDATA::Decode),
//	  Update=7 (@0x85836c: Decode4 partyId + PARTYDATA::Decode).
//	  The shared encoders carry only GMS>=87 (Invite jobId/level) and GMS>=95
//	  (member_data / town_portal) gates — both OFF for v61 and v83 — so each v61
//	  encode is asserted byte-equal to the IDA-verified v83 encode (cross-version
//	  equality, the door/SpawnDoor discipline).
//
// CannotKick (v83 case 25) is VERSION-ABSENT in v61: the v61 switch has no case
// 25/0x19 (upper block runs 0x1A/0x1B/0x1C/0x1D/0x1F/0x20 = 26/27/28/29/31/32,
// no kick arm) — same as v72. It gets no v61 fixture/marker/evidence.

// packet-audit:verify packet=party/clientbound/PartyAlreadyJoined1 version=gms_v61 ida=0x857a8c
// packet-audit:verify packet=party/clientbound/PartyBeginnerCannotCreate version=gms_v61 ida=0x857a8c
// packet-audit:verify packet=party/clientbound/PartyNotInParty version=gms_v61 ida=0x857a8c
// packet-audit:verify packet=party/clientbound/PartyAlreadyJoined2 version=gms_v61 ida=0x857a8c
// packet-audit:verify packet=party/clientbound/PartyPartyFull version=gms_v61 ida=0x857a8c
// packet-audit:verify packet=party/clientbound/PartyUnableToFindInChannel version=gms_v61 ida=0x857a8c
// packet-audit:verify packet=party/clientbound/PartyOnlyWithinVicinity version=gms_v61 ida=0x857a8c
// packet-audit:verify packet=party/clientbound/PartyUnableToHandOver version=gms_v61 ida=0x857a8c
// packet-audit:verify packet=party/clientbound/PartyOnlySameChannel version=gms_v61 ida=0x857a8c
// packet-audit:verify packet=party/clientbound/PartyGmCannotCreate version=gms_v61 ida=0x857a8c
// packet-audit:verify packet=party/clientbound/PartyUnableToFindCharacter version=gms_v61 ida=0x857a8c
func TestPartyModeOnlyArmsV61(t *testing.T) {
	cases := map[byte][]byte{
		9:  NewAlreadyJoined1(9).Encode(nil, nil)(nil),
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
			t.Errorf("v61 party mode-only mode %d: got % x want %02x", mode, got, mode)
		}
	}
}

// packet-audit:verify packet=party/clientbound/PartyBlockingInvitations version=gms_v61 ida=0x857a8c
// packet-audit:verify packet=party/clientbound/PartyTakingCareOfInvitation version=gms_v61 ida=0x857a8c
// packet-audit:verify packet=party/clientbound/PartyRequestDenied version=gms_v61 ida=0x857a8c
func TestPartyNameArmsV61(t *testing.T) {
	// WriteByte(mode) + WriteAsciiString("Bob") = [mode, 03 00, 'B','o','b'].
	bob := []byte{0x03, 0x00, 0x42, 0x6F, 0x62}
	want := func(mode byte) []byte { return append([]byte{mode}, bob...) }
	if got := NewBlockingInvitations(21, "Bob").Encode(nil, nil)(nil); !bytes.Equal(got, want(21)) {
		t.Errorf("v61 BlockingInvitations: got % x want % x", got, want(21))
	}
	if got := NewTakingCareOfInvitation(22, "Bob").Encode(nil, nil)(nil); !bytes.Equal(got, want(22)) {
		t.Errorf("v61 TakingCareOfInvitation: got % x want % x", got, want(22))
	}
	if got := NewRequestDenied(23, "Bob").Encode(nil, nil)(nil); !bytes.Equal(got, want(23)) {
		t.Errorf("v61 RequestDenied: got % x want % x", got, want(23))
	}
}

// packet-audit:verify packet=party/clientbound/PartyCreated version=gms_v61 ida=0x857a8c
// packet-audit:verify packet=party/clientbound/PartyChangeLeader version=gms_v61 ida=0x857a8c
// packet-audit:verify packet=party/clientbound/PartyDisband version=gms_v61 ida=0x857a8c
// packet-audit:verify packet=party/clientbound/PartyInvite version=gms_v61 ida=0x857a8c
// packet-audit:verify packet=party/clientbound/PartyJoin version=gms_v61 ida=0x857a8c
// packet-audit:verify packet=party/clientbound/PartyLeft version=gms_v61 ida=0x857a8c
// packet-audit:verify packet=party/clientbound/PartyUpdate version=gms_v61 ida=0x857a8c
func TestPartyDataArmsV61(t *testing.T) {
	v61 := pt.CreateContext("GMS", 61, 1)
	v83 := pt.CreateContext("GMS", 83, 1)
	members := []party.PartyMember{
		{Id: 100, Name: "Player1", JobId: 111, Level: 50, ChannelId: 1, MapId: 100000},
		{Id: 200, Name: "Player2", JobId: 222, Level: 70, ChannelId: -2, MapId: 200000},
		{Id: 300, Name: "Player3", JobId: 333, Level: 90, ChannelId: 3, MapId: 300000},
	}
	type arm struct {
		name     string
		v61, v83 []byte
	}
	// Modes are the real v61 switch case bytes (== v72): Created=8, ChangeLeader=26,
	// Disband=12, Invite=4, Join=15, Left=12, Update=7. The equality check is
	// mode-agnostic (same mode on both sides); the real value is the body being
	// version-stable across GMS<87 (Invite) and GMS<95 (member list).
	arms := []arm{
		{"Created", NewCreated(8, 12345).Encode(nil, v61)(nil), NewCreated(8, 12345).Encode(nil, v83)(nil)},
		{"ChangeLeader", NewChangeLeader(26, 9999, true).Encode(nil, v61)(nil), NewChangeLeader(26, 9999, true).Encode(nil, v83)(nil)},
		{"Disband", NewDisband(12, 5000, 300).Encode(nil, v61)(nil), NewDisband(12, 5000, 300).Encode(nil, v83)(nil)},
		{"Invite", NewInvite(4, 5000, "PartyLeader", 100, 50).Encode(nil, v61)(nil), NewInvite(4, 5000, "PartyLeader", 100, 50).Encode(nil, v83)(nil)},
		{"Join", NewJoin(15, 5000, "Player2", members, 100).Encode(nil, v61)(nil), NewJoin(15, 5000, "Player2", members, 100).Encode(nil, v83)(nil)},
		{"Left", NewLeft(12, 5000, 100, "Player1", false, members, 200).Encode(nil, v61)(nil), NewLeft(12, 5000, 100, "Player1", false, members, 200).Encode(nil, v83)(nil)},
		{"Update", NewUpdate(7, 5000, members, 100).Encode(nil, v61)(nil), NewUpdate(7, 5000, members, 100).Encode(nil, v83)(nil)},
	}
	for _, a := range arms {
		if !bytes.Equal(a.v61, a.v83) {
			t.Errorf("%s v61 != v83\n v61: % x\n v83: % x", a.name, a.v61, a.v83)
		}
	}
}
