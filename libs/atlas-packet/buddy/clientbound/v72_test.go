package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v72 BUDDYLIST (op 60) family verification — CWvsContext::OnFriendResult
// @0x935ecf, switch(Decode1(mode)) @0x935efa (GMS_v72.1_U_DEVM.exe, port 13339).
//
// The v72 OnFriendResult mode table is BYTE-IDENTICAL to v79/v83 (the buddy mode
// table is not shifted across GMS versions). Every case byte was read directly
// from the v72 switch @0x935efa:
//
//	error mode-only arms — case 0xB/0xC/0xD/0xE/0xF (StringPool notice, no read):
//	  ListFull=11 @0x936223 (StringPool 720), OtherListFull=12 @0x936237 (721),
//	  AlreadyBuddy=13 @0x93624b (722), CannotBuddyGm=14 @0x936273 (724),
//	  CharacterNotFound=15 @0x93625f (723). Wire = [mode].
//	UnknownError arms — case 0x10/0x11/0x13/0x16 @0x9361bd:
//	  `if(Decode1(v2)) DecodeStr+Notice else StringPool719` → reads one trailing
//	  flag byte in GMS. Modes 16/17/19/22. Wire = [mode, 0] (GMS no-name path).
//	data arms (read order verified in the v72 case bodies + helpers):
//	  Update=8       — CFriend::UpdateFriend @0x936c94: Decode4(charId)
//	                   @0x936c9c + GW_Friend::Decode(39) @0x936cc1 + Decode1(inShop)
//	                   @0x936cd7.
//	  ListUpdate=7   — sub_936B72 @0x936b72: Decode1(count) @0x936b83 +
//	                   DecodeBuffer(39*n) @0x936bd2 + DecodeBuffer(4*n) @0x936be2.
//	                   (cases 0x7/0xA/0x12 all route to this same reader @0x935f08.)
//	  Invite=9       — case 9 @0x9360d1: Decode4(originatorId) @0x9360d5 +
//	                   DecodeStr(name) @0x9360ec + sub_936C0F @0x936111
//	                   (GW_Friend::Decode(39) @0x936c56 + Decode1(inShop) @0x936c66);
//	                   NO jobId/level on v72 (<87), matching the v83 path.
//	  ChannelChange=20 — case 0x14 @0x935f14: Decode4(charId) @0x935f14 +
//	                   Decode1(inShop) @0x935f3b + Decode4(channel) @0x935f3d.
//	  CapacityUpdate=21 — case 0x15 @0x936188: Decode1(capacity) @0x936198.
//	GW_Friend::Decode @0x4d08d7 is a flat DecodeBuffer(39) opaque record; the
//	data-arm bodies are version-stable for GMS<87, so each v72 encode is asserted
//	byte-equal to the IDA-verified v83 encode (cross-version equality, the
//	door/SpawnDoor discipline). BuddyAlreadyBuddy (case 13) is covered by
//	TestBuddyAlreadyBuddyV72 in error_test.go.

// packet-audit:verify packet=buddy/clientbound/BuddyListFull version=gms_v72 ida=0x935ecf
// packet-audit:verify packet=buddy/clientbound/BuddyOtherListFull version=gms_v72 ida=0x935ecf
// packet-audit:verify packet=buddy/clientbound/BuddyCannotBuddyGm version=gms_v72 ida=0x935ecf
// packet-audit:verify packet=buddy/clientbound/BuddyCharacterNotFound version=gms_v72 ida=0x935ecf
func TestBuddyModeOnlyArmsV72(t *testing.T) {
	cases := map[byte]func(byte) []byte{
		11: func(b byte) []byte { return NewListFull(b).Encode(nil, nil)(nil) },
		12: func(b byte) []byte { return NewOtherListFull(b).Encode(nil, nil)(nil) },
		14: func(b byte) []byte { return NewCannotBuddyGm(b).Encode(nil, nil)(nil) },
		15: func(b byte) []byte { return NewCharacterNotFound(b).Encode(nil, nil)(nil) },
	}
	for mode, enc := range cases {
		if got := enc(mode); !bytes.Equal(got, []byte{mode}) {
			t.Errorf("v72 mode-only arm mode %d: got % x want %02x", mode, got, mode)
		}
	}
}

// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError version=gms_v72 ida=0x935ecf
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError2 version=gms_v72 ida=0x935ecf
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError3 version=gms_v72 ida=0x935ecf
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError4 version=gms_v72 ida=0x935ecf
func TestBuddyUnknownErrorArmsV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	cases := map[byte][]byte{
		16: NewUnknownError(16).Encode(nil, ctx)(nil),
		17: NewUnknownError2(17).Encode(nil, ctx)(nil),
		19: NewUnknownError3(19).Encode(nil, ctx)(nil),
		22: NewUnknownError4(22).Encode(nil, ctx)(nil),
	}
	for mode, got := range cases {
		if want := []byte{mode, 0x00}; !bytes.Equal(got, want) {
			t.Errorf("v72 UnknownError mode %d: got % x want % x", mode, got, want)
		}
	}
}

// packet-audit:verify packet=buddy/clientbound/BuddyUpdate version=gms_v72 ida=0x935ecf
// packet-audit:verify packet=buddy/clientbound/BuddyListUpdate version=gms_v72 ida=0x935ecf
// packet-audit:verify packet=buddy/clientbound/BuddyInvite version=gms_v72 ida=0x935ecf
// packet-audit:verify packet=buddy/clientbound/BuddyChannelChange version=gms_v72 ida=0x935ecf
// packet-audit:verify packet=buddy/clientbound/BuddyCapacityUpdate version=gms_v72 ida=0x935ecf
func TestBuddyDataArmsV72(t *testing.T) {
	v72 := pt.CreateContext("GMS", 72, 1)
	v83 := pt.CreateContext("GMS", 83, 1)

	// Update (mode 8): charId + GW_Friend(39) + inShop.
	up := NewBuddyUpdate(8, 1000, "TestPlayer", "Default Group", 1, false)
	if a, b := up.Encode(nil, v72)(nil), up.Encode(nil, v83)(nil); !bytes.Equal(a, b) {
		t.Errorf("Update v72 != v83\n v72: % x\n v83: % x", a, b)
	}
	// ListUpdate (mode 7): count + n*GW_Friend(39) + n*inShop(4).
	lu := NewBuddyListUpdate(7, []BuddyEntry{
		{CharacterId: 1000, Name: "Player1", ChannelId: 1, Group: "Default Group", InShop: false},
		{CharacterId: 2000, Name: "Player2", ChannelId: 2, Group: "Friends", InShop: true},
	})
	if a, b := lu.Encode(nil, v72)(nil), lu.Encode(nil, v83)(nil); !bytes.Equal(a, b) {
		t.Errorf("ListUpdate v72 != v83\n v72: % x\n v83: % x", a, b)
	}
	// Invite (mode 9): originatorId + name + GW_Friend(39) + inShop; NO jobId/level on v72 (<87).
	inv := NewBuddyInvite(9, 1000, 2000, "TestPlayer", 510, 120)
	if a, b := inv.Encode(nil, v72)(nil), inv.Encode(nil, v83)(nil); !bytes.Equal(a, b) {
		t.Errorf("Invite v72 != v83\n v72: % x\n v83: % x", a, b)
	}
	if got := inv.Encode(nil, v72)(nil); len(got) != 57 {
		t.Errorf("Invite v72 length: got %d want 57 (no jobId/level for <87)", len(got))
	}
	// ChannelChange (mode 20): charId + inShop(1) + channel(4).
	cc := NewBuddyChannelChange(20, 1000, 3)
	if a, b := cc.Encode(nil, v72)(nil), cc.Encode(nil, v83)(nil); !bytes.Equal(a, b) {
		t.Errorf("ChannelChange v72 != v83\n v72: % x\n v83: % x", a, b)
	}
	// CapacityUpdate (mode 21): capacity byte.
	cu := NewBuddyCapacityUpdate(21, 50)
	if got := cu.Encode(nil, v72)(nil); !bytes.Equal(got, []byte{21, 50}) {
		t.Errorf("CapacityUpdate v72: got % x want 15 32", got)
	}
}
