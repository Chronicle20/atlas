package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v61 BUDDYLIST (op 60) family verification — CWvsContext::OnFriendResult
// @0x85898e, switch(Decode1(mode)) @0x8589b9 (GMS_v61.1_U_DEVM.exe, port 13338).
// CWvsContext::OnPacket @0x8303eb case 60 @0x8304bd → OnFriendResult @0x85898e.
//
// Every case byte in the v61 switch is BYTE-IDENTICAL to the IDA-verified v72
// OnFriendResult switch (the buddy mode table is not shifted across GMS
// versions); each case body was read directly from the v61 decompile:
//
//	list-reset/list-update — case 0x7/0xA/0x12 @0x8589c7 → sub_859408 @0x859408:
//	  Decode1(count) @0x859419 + DecodeBuffer(22*n) @0x859468 + DecodeBuffer(4*n)
//	  @0x859478. ListUpdate=7.
//	Update — case 8 @0x858c40 → CFriend::UpdateFriend @0x85952a: Decode4(charId)
//	  @0x859532 + GW_Friend::Decode(22) @0x859557 + Decode1(inShop) @0x85956d.
//	Invite — case 9 @0x858b90: Decode4(originatorId) @0x858b94 + DecodeStr(name)
//	  @0x858bab + sub_8594A5 @0x8594a5 (GW_Friend::Decode(22) @0x8594ec +
//	  Decode1(inShop) @0x8594fc); NO jobId/level on v61 (<87), matching v83.
//	error mode-only arms — case 0xB/0xC/0xD/0xE/0xF (sub_678022 StringPool +
//	  CUtilDlg::Notice, NO wire read): ListFull=11 @0x858ced (SP727),
//	  OtherListFull=12 @0x858d01 (SP728), AlreadyBuddy=13 @0x858d15 (SP729),
//	  CannotBuddyGm=14 @0x858d43 (SP731), CharacterNotFound=15 @0x858d29 (SP730).
//	  Wire = [mode]. (StringPool ids differ from v72's 720-724 — an off-wire
//	  table renumbering, ignored.)
//	UnknownError arms — case 0x10/0x11/0x13/0x16 @0x858c7c: `if (Decode1(v2))
//	  DecodeStr+Notice else StringPool726` → reads one trailing flag byte in GMS.
//	  Modes 16/17/19/22. Wire = [mode, 0] (GMS no-name path).
//	ChannelChange — case 0x14 @0x8589d3: Decode4(charId) @0x8589d3 + Decode1(inShop)
//	  @0x8589fa + Decode4(channel) @0x8589fc. No GW_Friend record → version-stable.
//	CapacityUpdate — case 0x15 @0x858c57: Decode1(capacity) @0x858c57.
//
// GW_Friend::Decode @0x4b54d8 is DecodeBuffer(this, 22) — a 22-byte record
// (FriendId(4)+FriendName(13)+Flag(1)+ChannelId(4)), 17 bytes SHORTER than the
// v72+ 39-byte record because the trailing FriendGroup postdates v61. This
// divergence was fixed first (BuddyHasFriendGroup gate, prior commit); the
// data-arm fixtures below assert the group-less v61 wire explicitly.

// packet-audit:verify packet=buddy/clientbound/BuddyListFull version=gms_v61 ida=0x85898e
// packet-audit:verify packet=buddy/clientbound/BuddyOtherListFull version=gms_v61 ida=0x85898e
// packet-audit:verify packet=buddy/clientbound/BuddyCannotBuddyGm version=gms_v61 ida=0x85898e
// packet-audit:verify packet=buddy/clientbound/BuddyCharacterNotFound version=gms_v61 ida=0x85898e
func TestBuddyModeOnlyArmsV61(t *testing.T) {
	cases := map[byte]func(byte) []byte{
		11: func(b byte) []byte { return NewListFull(b).Encode(nil, nil)(nil) },
		12: func(b byte) []byte { return NewOtherListFull(b).Encode(nil, nil)(nil) },
		14: func(b byte) []byte { return NewCannotBuddyGm(b).Encode(nil, nil)(nil) },
		15: func(b byte) []byte { return NewCharacterNotFound(b).Encode(nil, nil)(nil) },
	}
	for mode, enc := range cases {
		if got := enc(mode); !bytes.Equal(got, []byte{mode}) {
			t.Errorf("v61 mode-only arm mode %d: got % x want %02x", mode, got, mode)
		}
	}
}

// AlreadyBuddy (case 0xD @0x858d15) is mode-only in v61: sub_678022(&v28, 729) +
// CUtilDlg::Notice, no wire read after the mode byte. v61 case byte = 13, same
// as v72/v79/v83.
//
// packet-audit:verify packet=buddy/clientbound/BuddyAlreadyBuddy version=gms_v61 ida=0x85898e
func TestBuddyAlreadyBuddyV61(t *testing.T) {
	const v61Mode = 13
	got := NewAlreadyBuddy(v61Mode).Encode(nil, nil)(nil)
	if want := []byte{v61Mode}; !bytes.Equal(got, want) {
		t.Fatalf("v61 BuddyAlreadyBuddy: got %v want %v", got, want)
	}
}

// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError version=gms_v61 ida=0x85898e
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError2 version=gms_v61 ida=0x85898e
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError3 version=gms_v61 ida=0x85898e
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError4 version=gms_v61 ida=0x85898e
func TestBuddyUnknownErrorArmsV61(t *testing.T) {
	ctx := pt.CreateContext("GMS", 61, 1)
	cases := map[byte][]byte{
		16: NewUnknownError(16).Encode(nil, ctx)(nil),
		17: NewUnknownError2(17).Encode(nil, ctx)(nil),
		19: NewUnknownError3(19).Encode(nil, ctx)(nil),
		22: NewUnknownError4(22).Encode(nil, ctx)(nil),
	}
	for mode, got := range cases {
		if want := []byte{mode, 0x00}; !bytes.Equal(got, want) {
			t.Errorf("v61 UnknownError mode %d: got % x want % x", mode, got, want)
		}
	}
}

// Data arms. Update/ListUpdate/Invite carry GW_Friend records that are 22 bytes
// in v61 (no FriendGroup); their expected bytes are hand-computed from the v61
// read order. ChannelChange/CapacityUpdate carry no GW_Friend record and are
// byte-identical to the IDA-verified v83 encode (cross-version equality).
//
// packet-audit:verify packet=buddy/clientbound/BuddyUpdate version=gms_v61 ida=0x85898e
// packet-audit:verify packet=buddy/clientbound/BuddyListUpdate version=gms_v61 ida=0x85898e
// packet-audit:verify packet=buddy/clientbound/BuddyInvite version=gms_v61 ida=0x85898e
// packet-audit:verify packet=buddy/clientbound/BuddyChannelChange version=gms_v61 ida=0x85898e
// packet-audit:verify packet=buddy/clientbound/BuddyCapacityUpdate version=gms_v61 ida=0x85898e
func TestBuddyDataArmsV61(t *testing.T) {
	v61 := pt.CreateContext("GMS", 61, 1)
	v83 := pt.CreateContext("GMS", 83, 1)

	// Update (mode 8): mode + charId(4) + GW_Friend(22, no group) + inShop(1).
	// NewBuddyUpdate(8, 1000, "TestPlayer", "Default Group", 1, false).
	updateWant := []byte{
		0x08,                   // mode
		0xE8, 0x03, 0x00, 0x00, // characterId 1000
		0xE8, 0x03, 0x00, 0x00, // GW_Friend.FriendId 1000
		0x54, 0x65, 0x73, 0x74, 0x50, 0x6C, 0x61, 0x79, 0x65, 0x72, 0x00, 0x00, 0x00, // "TestPlayer" (13)
		0x00,                   // flag
		0x01, 0x00, 0x00, 0x00, // channelId 1
		0x00, // inShop false
	}
	if got := NewBuddyUpdate(8, 1000, "TestPlayer", "Default Group", 1, false).Encode(nil, v61)(nil); !bytes.Equal(got, updateWant) {
		t.Errorf("v61 Update:\n got % x\nwant % x", got, updateWant)
	}

	// Invite (mode 9): mode + originatorId(4) + name(2+10) + GW_Friend(22) + inShop(1).
	// No jobId/level (<87). NewBuddyInvite(9, actor 1000, originator 2000, "TestPlayer", 510, 120).
	inviteWant := []byte{
		0x09,                   // mode
		0xD0, 0x07, 0x00, 0x00, // originatorId 2000
		0x0A, 0x00, 0x54, 0x65, 0x73, 0x74, 0x50, 0x6C, 0x61, 0x79, 0x65, 0x72, // AsciiString "TestPlayer"
		0xE8, 0x03, 0x00, 0x00, // GW_Friend.FriendId = actorId 1000
		0x54, 0x65, 0x73, 0x74, 0x50, 0x6C, 0x61, 0x79, 0x65, 0x72, 0x00, 0x00, 0x00, // friendName "TestPlayer" (13)
		0x00,                   // flag
		0x00, 0x00, 0x00, 0x00, // channelId 0
		0x00, // inShop false
	}
	if got := NewBuddyInvite(9, 1000, 2000, "TestPlayer", 510, 120).Encode(nil, v61)(nil); !bytes.Equal(got, inviteWant) {
		t.Errorf("v61 Invite:\n got % x\nwant % x", got, inviteWant)
	}

	// ListUpdate (mode 7): mode + count(1) + n*GW_Friend(22) + n*inShop(4).
	listWant := []byte{
		0x07, // mode
		0x02, // count
		// buddy0: id 1000, "Player1", flag 0, channel 1
		0xE8, 0x03, 0x00, 0x00,
		0x50, 0x6C, 0x61, 0x79, 0x65, 0x72, 0x31, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00,
		0x01, 0x00, 0x00, 0x00,
		// buddy1: id 2000, "Player2", flag 0, channel 2
		0xD0, 0x07, 0x00, 0x00,
		0x50, 0x6C, 0x61, 0x79, 0x65, 0x72, 0x32, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00,
		0x02, 0x00, 0x00, 0x00,
		// inShop ints (uint32): buddy0 false, buddy1 true
		0x00, 0x00, 0x00, 0x00,
		0x01, 0x00, 0x00, 0x00,
	}
	lu := NewBuddyListUpdate(7, []BuddyEntry{
		{CharacterId: 1000, Name: "Player1", ChannelId: 1, Group: "Default Group", InShop: false},
		{CharacterId: 2000, Name: "Player2", ChannelId: 2, Group: "Friends", InShop: true},
	})
	if got := lu.Encode(nil, v61)(nil); !bytes.Equal(got, listWant) {
		t.Errorf("v61 ListUpdate:\n got % x\nwant % x", got, listWant)
	}

	// ChannelChange (mode 20): charId + inShop(1) + channel(4). No GW_Friend → == v83.
	cc := NewBuddyChannelChange(20, 1000, 3)
	if a, b := cc.Encode(nil, v61)(nil), cc.Encode(nil, v83)(nil); !bytes.Equal(a, b) {
		t.Errorf("v61 ChannelChange != v83\n v61: % x\n v83: % x", a, b)
	}

	// CapacityUpdate (mode 21): capacity byte. NewBuddyCapacityUpdate(21, 50).
	if got := NewBuddyCapacityUpdate(21, 50).Encode(nil, v61)(nil); !bytes.Equal(got, []byte{21, 50}) {
		t.Errorf("v61 CapacityUpdate: got % x want 15 32", got)
	}
}
