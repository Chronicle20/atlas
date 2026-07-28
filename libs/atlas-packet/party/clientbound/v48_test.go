package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/party"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v48 PARTY_OPERATION (clientbound) family verification —
// CWvsContext::OnPartyResult @0x729935, switch(Decode1(mode)) (GMS_v48_1_DEVM.exe,
// port 13337). Every case byte and body below was read directly from the v48
// decompile (task-113 v48 close-I; the close-H StringPool decrypt maps each
// notice arm's mode → SP id). The v48 mode table is RE-PACKED vs v61 (different
// case bytes) and three data arms DIVERGE from v61/v83 on the wire:
//
//	mode-only notice arms (case → StringPool id, NO further wire read):
//	  AlreadyJoined1=8 (SP310), BeginnerCannotCreate=9 (SP311), NotInParty=12
//	  (SP312), AlreadyJoined2=15 (SP310), PartyFull=16 (SP313),
//	  UnableToFindInChannel=17 (SP319), GmCannotCreate=24 (SP2573),
//	  UnableToFindCharacter=25 (SP355). Each case is sub_5D75AF(id)+ChatLogAdd
//	  and reads NOTHING further off the wire → mode-only ([mode]).
//	name arms (case → DecodeStr(target) then StringPool notice; wire=[mode,name]):
//	  BlockingInvitations=19 (SP299), TakingCareOfInvitation=20 (SP2435),
//	  RequestDenied=21 (SP300).
//	data arms (read order verified in the case bodies @0x729935):
//	  Invite=4     Decode4(partyId)+DecodeStr(name)          DIVERGES: NO autoJoin byte
//	  Update=6/26  Decode4(partyId)+PARTYDATA::Decode(294)   DIVERGES: PARTYDATA has NO leaderId
//	  Created=7    Decode4(partyId)+Decode4+Decode4+Decode2+Decode2   = v83
//	  Left=11(t)   Decode4(pid)+Decode4(tid)+Decode1(=1)+Decode1(forced)+DecodeStr+PARTYDATA(294)  DIVERGES: no leaderId
//	  Disband=11(e) Decode4(pid)+Decode4(tid)+Decode1(=0), STOPS       DIVERGES: NO trailing partyId
//	  Join=14      Decode4(partyId)+DecodeStr(name)+PARTYDATA(294)     DIVERGES: no leaderId
//	  TownPortal=29 Decode1(slot)+Decode4(town)+Decode4(field)+Decode2(x)+Decode2(y)  = v83 (no v95 skillId)
//
// PARTYDATA::Decode @0x49c925 = DecodeBuffer(0x126=294); v61/v83 = 298. The
// 4-byte delta is exactly the leaderId the v61+ struct inserts after the channel
// array — absent here. IDA-verified: OnPartyResult qmemcpy/memset both use 294.
//
// v48-ABSENT arms (no case decodes to them → no fixture/marker/evidence):
//   ChangeLeader (folds into the Update PARTYDATA body, case 6/26 — no discrete
//   Decode4+Decode1 change-leader arm), CannotKick, OnlyWithinVicinity,
//   UnableToHandOver, OnlySameChannel (leadership-transfer/kick notices — no v48
//   case). MemberHP (case 27) is a SEPARATE writer (UPDATE_PARTYMEMBER_HP), out
//   of this op-cell. InviteReject (case 28) has no atlas party arm.

// packet-audit:verify packet=party/clientbound/PartyAlreadyJoined1 version=gms_v48 ida=0x729935
// packet-audit:verify packet=party/clientbound/PartyBeginnerCannotCreate version=gms_v48 ida=0x729935
// packet-audit:verify packet=party/clientbound/PartyNotInParty version=gms_v48 ida=0x729935
// packet-audit:verify packet=party/clientbound/PartyAlreadyJoined2 version=gms_v48 ida=0x729935
// packet-audit:verify packet=party/clientbound/PartyPartyFull version=gms_v48 ida=0x729935
// packet-audit:verify packet=party/clientbound/PartyUnableToFindInChannel version=gms_v48 ida=0x729935
// packet-audit:verify packet=party/clientbound/PartyGmCannotCreate version=gms_v48 ida=0x729935
// packet-audit:verify packet=party/clientbound/PartyUnableToFindCharacter version=gms_v48 ida=0x729935
func TestPartyModeOnlyArmsV48(t *testing.T) {
	cases := map[byte][]byte{
		8:  NewAlreadyJoined1(8).Encode(nil, nil)(nil),
		9:  NewBeginnerCannotCreate(9).Encode(nil, nil)(nil),
		12: NewNotInParty(12).Encode(nil, nil)(nil),
		15: NewAlreadyJoined2(15).Encode(nil, nil)(nil),
		16: NewPartyFull(16).Encode(nil, nil)(nil),
		17: NewUnableToFindInChannel(17).Encode(nil, nil)(nil),
		24: NewGmCannotCreate(24).Encode(nil, nil)(nil),
		25: NewUnableToFindCharacter(25).Encode(nil, nil)(nil),
	}
	for mode, got := range cases {
		if !bytes.Equal(got, []byte{mode}) {
			t.Errorf("v48 party mode-only mode %d: got % x want %02x", mode, got, mode)
		}
	}
}

// packet-audit:verify packet=party/clientbound/PartyBlockingInvitations version=gms_v48 ida=0x729935
// packet-audit:verify packet=party/clientbound/PartyTakingCareOfInvitation version=gms_v48 ida=0x729935
// packet-audit:verify packet=party/clientbound/PartyRequestDenied version=gms_v48 ida=0x729935
func TestPartyNameArmsV48(t *testing.T) {
	// WriteByte(mode) + WriteAsciiString("Bob") = [mode, 03 00, 'B','o','b'].
	bob := []byte{0x03, 0x00, 0x42, 0x6F, 0x62}
	want := func(mode byte) []byte { return append([]byte{mode}, bob...) }
	if got := NewBlockingInvitations(19, "Bob").Encode(nil, nil)(nil); !bytes.Equal(got, want(19)) {
		t.Errorf("v48 BlockingInvitations: got % x want % x", got, want(19))
	}
	if got := NewTakingCareOfInvitation(20, "Bob").Encode(nil, nil)(nil); !bytes.Equal(got, want(20)) {
		t.Errorf("v48 TakingCareOfInvitation: got % x want % x", got, want(20))
	}
	if got := NewRequestDenied(21, "Bob").Encode(nil, nil)(nil); !bytes.Equal(got, want(21)) {
		t.Errorf("v48 RequestDenied: got % x want % x", got, want(21))
	}
}

// removeBytes returns b with n bytes removed starting at off.
func removeBytes(b []byte, off, n int) []byte {
	out := make([]byte, 0, len(b)-n)
	out = append(out, b[:off]...)
	out = append(out, b[off+n:]...)
	return out
}

// TestPartyDataArmsV48 verifies each data arm's v48 wire body against the
// IDA-derived divergences. The divergent arms are checked as an exact transform
// of the (already-verified) v83 encode — proving both the divergence and that
// nothing else shifted. The non-divergent arms are asserted byte-equal to v83.
// packet-audit:verify packet=party/clientbound/PartyCreated version=gms_v48 ida=0x729935
// packet-audit:verify packet=party/clientbound/PartyInvite version=gms_v48 ida=0x729935
// packet-audit:verify packet=party/clientbound/PartyJoin version=gms_v48 ida=0x729935
// packet-audit:verify packet=party/clientbound/PartyLeft version=gms_v48 ida=0x729935
// packet-audit:verify packet=party/clientbound/PartyUpdate version=gms_v48 ida=0x729935
// packet-audit:verify packet=party/clientbound/PartyDisband version=gms_v48 ida=0x729935
// (TownPortal is asserted below for coverage but is not a tracked op-cell arm —
// no PartyTownPortal audit report exists — so it carries no packet-audit marker.)
func TestPartyDataArmsV48(t *testing.T) {
	v48 := pt.CreateContext("GMS", 48, 1)
	v83 := pt.CreateContext("GMS", 83, 1)
	members := []party.PartyMember{
		{Id: 100, Name: "Player1", JobId: 111, Level: 50, ChannelId: 1, MapId: 100000},
		{Id: 200, Name: "Player2", JobId: 222, Level: 70, ChannelId: -2, MapId: 200000},
		{Id: 300, Name: "Player3", JobId: 333, Level: 90, ChannelId: 3, MapId: 300000},
	}
	const partyDataLeaderOff = 174 // ids(24)+names(78)+jobs(24)+levels(24)+channels(24)

	// Created (mode 7) — no version gate, v48 == v83.
	if got, want := NewCreated(7, 12345).Encode(nil, v48)(nil), NewCreated(7, 12345).Encode(nil, v83)(nil); !bytes.Equal(got, want) {
		t.Errorf("Created v48 != v83\n v48: % x\n v83: % x", got, want)
	}

	// TownPortal (mode 29) — v95-gated skillId only, v48 == v83.
	if got, want := NewTownPortal(29, 2, 100000000, 100010000, -300, 150).Encode(nil, v48)(nil), NewTownPortal(29, 2, 100000000, 100010000, -300, 150).Encode(nil, v83)(nil); !bytes.Equal(got, want) {
		t.Errorf("TownPortal v48 != v83\n v48: % x\n v83: % x", got, want)
	}

	// Invite (mode 4) — v48 drops the trailing autoJoin byte (v83 appends it as 0).
	{
		got := NewInvite(4, 5000, "PartyLeader", 100, 50).Encode(nil, v48)(nil)
		v83b := NewInvite(4, 5000, "PartyLeader", 100, 50).Encode(nil, v83)(nil)
		if v83b[len(v83b)-1] != 0 {
			t.Fatalf("Invite v83 trailing byte expected autoJoin 0, got %02x", v83b[len(v83b)-1])
		}
		if want := v83b[:len(v83b)-1]; !bytes.Equal(got, want) {
			t.Errorf("Invite v48 != v83-minus-autoJoin\n v48: % x\n want: % x", got, want)
		}
		if len(got) != 18 { // mode(1)+partyId(4)+name(2+11)
			t.Errorf("Invite v48 byte count: got %d want 18", len(got))
		}
	}

	// Disband (mode 11 else-branch) — v48 stops after the const 0; v83 appends
	// the repeated partyId (last 4 bytes).
	{
		got := NewDisband(11, 5000, 300).Encode(nil, v48)(nil)
		v83b := NewDisband(11, 5000, 300).Encode(nil, v83)(nil)
		if want := v83b[:len(v83b)-4]; !bytes.Equal(got, want) {
			t.Errorf("Disband v48 != v83-minus-trailingPartyId\n v48: % x\n want: % x", got, want)
		}
		if len(got) != 10 { // mode(1)+partyId(4)+targetId(4)+const0(1)
			t.Errorf("Disband v48 byte count: got %d want 10", len(got))
		}
	}

	// Update (mode 6) — PARTYDATA has no leaderId; v48 == v83 with the 4-byte
	// leaderId removed at PARTYDATA offset 174 (arm prefix = mode+partyId = 5).
	{
		got := NewUpdate(6, 5000, members, 100).Encode(nil, v48)(nil)
		v83b := NewUpdate(6, 5000, members, 100).Encode(nil, v83)(nil)
		cut := 5 + partyDataLeaderOff
		if want := removeBytes(v83b, cut, 4); !bytes.Equal(got, want) {
			t.Errorf("Update v48 != v83-minus-leaderId\n v48: % x\n want: % x", got, want)
		}
		if len(got) != 299 { // 5 + 294
			t.Errorf("Update v48 byte count: got %d want 299", len(got))
		}
	}

	// Join (mode 14) — DecodeStr(name)+PARTYDATA(no leaderId).
	{
		got := NewJoin(14, 5000, "Player2", members, 100).Encode(nil, v48)(nil)
		v83b := NewJoin(14, 5000, "Player2", members, 100).Encode(nil, v83)(nil)
		cut := (1 + 4 + (2 + len("Player2"))) + partyDataLeaderOff
		if want := removeBytes(v83b, cut, 4); !bytes.Equal(got, want) {
			t.Errorf("Join v48 != v83-minus-leaderId\n v48: % x\n want: % x", got, want)
		}
	}

	// Left (mode 11 true-branch) — partyId+targetId+const1+forced+name+PARTYDATA(no leaderId).
	{
		got := NewLeft(11, 5000, 100, "Player1", false, members, 200).Encode(nil, v48)(nil)
		v83b := NewLeft(11, 5000, 100, "Player1", false, members, 200).Encode(nil, v83)(nil)
		cut := (1 + 4 + 4 + 1 + 1 + (2 + len("Player1"))) + partyDataLeaderOff
		if want := removeBytes(v83b, cut, 4); !bytes.Equal(got, want) {
			t.Errorf("Left v48 != v83-minus-leaderId\n v48: % x\n want: % x", got, want)
		}
	}
}

// TestPartyDataArmsV48RoundTrip proves the v48 legacy codecs are symmetric
// (encode → decode → re-encode) for the divergent data arms.
func TestPartyDataArmsV48RoundTrip(t *testing.T) {
	v48 := pt.CreateContext("GMS", 48, 1)
	members := []party.PartyMember{
		{Id: 100, Name: "Player1", JobId: 111, Level: 50, ChannelId: 1, MapId: 100000},
		{Id: 200, Name: "Player2", JobId: 222, Level: 70, ChannelId: -2, MapId: 200000},
	}

	// Invite
	{
		in := NewInvite(4, 5000, "PartyLeader", 100, 50)
		out := Invite{}
		pt.RoundTrip(t, v48, in.Encode, out.Decode, nil)
		if out.Mode() != in.Mode() || out.PartyId() != in.PartyId() || out.OriginatorName() != in.OriginatorName() {
			t.Errorf("Invite v48 round-trip mismatch: %+v vs %+v", out, in)
		}
	}
	// Disband
	{
		in := NewDisband(11, 5000, 300)
		out := Disband{}
		pt.RoundTrip(t, v48, in.Encode, out.Decode, nil)
		if out.Mode() != in.Mode() || out.PartyId() != in.PartyId() || out.TargetId() != in.TargetId() {
			t.Errorf("Disband v48 round-trip mismatch: %+v vs %+v", out, in)
		}
	}
	// Update (leaderId is not carried on the legacy wire → decodes to 0)
	{
		in := NewUpdate(6, 5000, members, 100)
		out := Update{}
		pt.RoundTrip(t, v48, in.Encode, out.Decode, nil)
		if out.Mode() != in.Mode() || out.PartyId() != in.PartyId() || len(out.Members()) != len(in.Members()) {
			t.Errorf("Update v48 round-trip mismatch: %+v vs %+v", out, in)
		}
		if out.LeaderId() != 0 {
			t.Errorf("Update v48 leaderId: got %d want 0 (not on legacy wire)", out.LeaderId())
		}
	}
	// Join
	{
		in := NewJoin(14, 5000, "Player2", members, 100)
		out := Join{}
		pt.RoundTrip(t, v48, in.Encode, out.Decode, nil)
		if out.Mode() != in.Mode() || out.TargetName() != in.TargetName() || len(out.Members()) != len(in.Members()) {
			t.Errorf("Join v48 round-trip mismatch: %+v vs %+v", out, in)
		}
	}
	// Left
	{
		in := NewLeft(11, 5000, 100, "Player1", true, members, 200)
		out := Left{}
		pt.RoundTrip(t, v48, in.Encode, out.Decode, nil)
		if out.Mode() != in.Mode() || out.TargetId() != in.TargetId() || out.Forced() != in.Forced() {
			t.Errorf("Left v48 round-trip mismatch: %+v vs %+v", out, in)
		}
	}
}
