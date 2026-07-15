package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/party"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v79 PARTY_OPERATION (op 59) family verification — CWvsContext::OnPartyResult
// @0x987583, switch(Decode1(mode)) @0x9875bd (GMS_v79_1_DEVM.exe, port 13340).
//
// The v79 OnPartyResult mode table is BYTE-IDENTICAL to v83 (v83/v84 are
// byte-identical; v87+ carry the non-uniform shift). Every case body below was
// read directly from the v79 decompile:
//
//	mode-only notice arms (case → StringPool id, NO further wire read):
//	  AlreadyJoined1=9(320), BeginnerCannotCreate=10(321), NotInParty=13(322),
//	  AlreadyJoined2=16(320), PartyFull=17(323), UnableToFindInChannel=19(332),
//	  CannotKick=25(5021), OnlyWithinVicinity=28(4027), UnableToHandOver=29(4029),
//	  OnlySameChannel=30(4028), GmCannotCreate=32(328), UnableToFindCharacter=33(368).
//	name arms (case 0x15/0x16/0x17 → DecodeStr(target) then StringPool notice):
//	  BlockingInvitations=21, TakingCareOfInvitation=22, RequestDenied=23. Wire = [mode, name].
//	data arms (PARTYDATA / member-list bodies, read order verified in the case
//	  bodies @0x987ee2/0x98794a/0x987ac7/0x987d9b etc.): Created, ChangeLeader,
//	  Disband, Invite, Join, Left, Update. These bodies are version-stable for
//	  GMS<87 (Invite jobId/level is gated >=87, off for both v79 and v83), so each
//	  v79 encode is asserted byte-equal to the IDA-verified v83 encode
//	  (cross-version equality, the door/SpawnDoor discipline).

// packet-audit:verify packet=party/clientbound/PartyAlreadyJoined2 version=gms_v79 ida=0x987583
// packet-audit:verify packet=party/clientbound/PartyBeginnerCannotCreate version=gms_v79 ida=0x987583
// packet-audit:verify packet=party/clientbound/PartyNotInParty version=gms_v79 ida=0x987583
// packet-audit:verify packet=party/clientbound/PartyPartyFull version=gms_v79 ida=0x987583
// packet-audit:verify packet=party/clientbound/PartyUnableToFindInChannel version=gms_v79 ida=0x987583
// packet-audit:verify packet=party/clientbound/PartyCannotKick version=gms_v79 ida=0x987583
// packet-audit:verify packet=party/clientbound/PartyOnlyWithinVicinity version=gms_v79 ida=0x987583
// packet-audit:verify packet=party/clientbound/PartyUnableToHandOver version=gms_v79 ida=0x987583
// packet-audit:verify packet=party/clientbound/PartyOnlySameChannel version=gms_v79 ida=0x987583
// packet-audit:verify packet=party/clientbound/PartyGmCannotCreate version=gms_v79 ida=0x987583
// packet-audit:verify packet=party/clientbound/PartyUnableToFindCharacter version=gms_v79 ida=0x987583
func TestPartyModeOnlyArmsV79(t *testing.T) {
	cases := map[byte][]byte{
		16: NewAlreadyJoined2(16).Encode(nil, nil)(nil),
		10: NewBeginnerCannotCreate(10).Encode(nil, nil)(nil),
		13: NewNotInParty(13).Encode(nil, nil)(nil),
		17: NewPartyFull(17).Encode(nil, nil)(nil),
		19: NewUnableToFindInChannel(19).Encode(nil, nil)(nil),
		25: NewCannotKick(25).Encode(nil, nil)(nil),
		28: NewOnlyWithinVicinity(28).Encode(nil, nil)(nil),
		29: NewUnableToHandOver(29).Encode(nil, nil)(nil),
		30: NewOnlySameChannel(30).Encode(nil, nil)(nil),
		32: NewGmCannotCreate(32).Encode(nil, nil)(nil),
		33: NewUnableToFindCharacter(33).Encode(nil, nil)(nil),
	}
	for mode, got := range cases {
		if !bytes.Equal(got, []byte{mode}) {
			t.Errorf("v79 party mode-only mode %d: got % x want %02x", mode, got, mode)
		}
	}
}

// packet-audit:verify packet=party/clientbound/PartyBlockingInvitations version=gms_v79 ida=0x987583
// packet-audit:verify packet=party/clientbound/PartyTakingCareOfInvitation version=gms_v79 ida=0x987583
// packet-audit:verify packet=party/clientbound/PartyRequestDenied version=gms_v79 ida=0x987583
func TestPartyNameArmsV79(t *testing.T) {
	// WriteByte(mode) + WriteAsciiString("Bob") = [mode, 03 00, 'B','o','b'].
	bob := []byte{0x03, 0x00, 0x42, 0x6F, 0x62}
	want := func(mode byte) []byte { return append([]byte{mode}, bob...) }
	if got := NewBlockingInvitations(21, "Bob").Encode(nil, nil)(nil); !bytes.Equal(got, want(21)) {
		t.Errorf("v79 BlockingInvitations: got % x want % x", got, want(21))
	}
	if got := NewTakingCareOfInvitation(22, "Bob").Encode(nil, nil)(nil); !bytes.Equal(got, want(22)) {
		t.Errorf("v79 TakingCareOfInvitation: got % x want % x", got, want(22))
	}
	if got := NewRequestDenied(23, "Bob").Encode(nil, nil)(nil); !bytes.Equal(got, want(23)) {
		t.Errorf("v79 RequestDenied: got % x want % x", got, want(23))
	}
}

// packet-audit:verify packet=party/clientbound/PartyCreated version=gms_v79 ida=0x987583
// packet-audit:verify packet=party/clientbound/PartyChangeLeader version=gms_v79 ida=0x987583
// packet-audit:verify packet=party/clientbound/PartyDisband version=gms_v79 ida=0x987583
// packet-audit:verify packet=party/clientbound/PartyInvite version=gms_v79 ida=0x987583
// packet-audit:verify packet=party/clientbound/PartyJoin version=gms_v79 ida=0x987583
// packet-audit:verify packet=party/clientbound/PartyLeft version=gms_v79 ida=0x987583
// packet-audit:verify packet=party/clientbound/PartyUpdate version=gms_v79 ida=0x987583
func TestPartyDataArmsV79(t *testing.T) {
	v79 := pt.CreateContext("GMS", 79, 1)
	v83 := pt.CreateContext("GMS", 83, 1)
	members := []party.PartyMember{
		{Id: 100, Name: "Player1", JobId: 111, Level: 50, ChannelId: 1, MapId: 100000},
		{Id: 200, Name: "Player2", JobId: 222, Level: 70, ChannelId: -2, MapId: 200000},
		{Id: 300, Name: "Player3", JobId: 333, Level: 90, ChannelId: 3, MapId: 300000},
	}
	type arm struct {
		name string
		v79  []byte
		v83  []byte
	}
	arms := []arm{
		{"Created", NewCreated(7, 12345).Encode(nil, v79)(nil), NewCreated(7, 12345).Encode(nil, v83)(nil)},
		{"ChangeLeader", NewChangeLeader(14, 9999, true).Encode(nil, v79)(nil), NewChangeLeader(14, 9999, true).Encode(nil, v83)(nil)},
		{"Disband", NewDisband(11, 5000, 300).Encode(nil, v79)(nil), NewDisband(11, 5000, 300).Encode(nil, v83)(nil)},
		{"Invite", NewInvite(16, 5000, "PartyLeader", 100, 50).Encode(nil, v79)(nil), NewInvite(16, 5000, "PartyLeader", 100, 50).Encode(nil, v83)(nil)},
		{"Join", NewJoin(12, 5000, "Player2", members, 100).Encode(nil, v79)(nil), NewJoin(12, 5000, "Player2", members, 100).Encode(nil, v83)(nil)},
		{"Left", NewLeft(10, 5000, 100, "Player1", false, members, 200).Encode(nil, v79)(nil), NewLeft(10, 5000, 100, "Player1", false, members, 200).Encode(nil, v83)(nil)},
		{"Update", NewUpdate(13, 5000, members, 100).Encode(nil, v79)(nil), NewUpdate(13, 5000, members, 100).Encode(nil, v83)(nil)},
	}
	for _, a := range arms {
		if !bytes.Equal(a.v79, a.v83) {
			t.Errorf("%s v79 != v83\n v79: % x\n v83: % x", a.name, a.v79, a.v83)
		}
	}
}
