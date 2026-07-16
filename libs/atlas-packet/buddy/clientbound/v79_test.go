package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v79 BUDDYLIST (op 60) family verification — CWvsContext::OnFriendResult
// @0x98854f, switch(Decode1(mode)) @0x98857a (GMS_v79_1_DEVM.exe, port 13340).
//
// The v79 OnFriendResult mode table is BYTE-IDENTICAL to v83 (the buddy mode
// table is not shifted across GMS versions). Each arm's case body was read from
// the v79 decompile:
//
//	error mode-only arms — case 0xB/0xC/0xD/0xE/0xF (StringPool notice, no read):
//	  ListFull=11, OtherListFull=12, AlreadyBuddy=13, CannotBuddyGm=14,
//	  CharacterNotFound=15. Wire = [mode].
//	UnknownError arms — case 0x10/0x11/0x13/0x16: `if(Decode1) DecodeStr+Notice
//	  else StringPool719` → reads one trailing flag byte in GMS. Modes 16/17/19/22.
//	  Wire = [mode, 0] (GMS no-name path).
//	data arms (read order verified in the v79 case bodies + helpers):
//	  Update=8       — UpdateFriend @0x989311: Decode4(charId)+GW_Friend(39)+Decode1.
//	  ListUpdate=7   — sub_9891EF @0x9891EF: Decode1(count)+DecodeBuffer(39*n)+DecodeBuffer(4*n).
//	  Invite=9       — case 9: Decode4(originatorId)+DecodeStr(name)+GW_Friend(39)+Decode1;
//	                   NO jobId/level on v79 (<87), matching the v83 path.
//	  ChannelChange=20 — case 0x14: Decode4(charId)+Decode1(inShop)+Decode4(channel).
//	  CapacityUpdate=21 — case 0x15: Decode1(capacity).
//	GW_Friend::Decode @0x4d86d2 is a flat DecodeBuffer(39) opaque record; the
//	data-arm bodies are version-stable for GMS<87, so each v79 encode is asserted
//	byte-equal to the IDA-verified v83 encode (cross-version equality, the
//	door/SpawnDoor discipline).

// packet-audit:verify packet=buddy/clientbound/BuddyListFull version=gms_v79 ida=0x98854f
// packet-audit:verify packet=buddy/clientbound/BuddyOtherListFull version=gms_v79 ida=0x98854f
// packet-audit:verify packet=buddy/clientbound/BuddyCannotBuddyGm version=gms_v79 ida=0x98854f
// packet-audit:verify packet=buddy/clientbound/BuddyCharacterNotFound version=gms_v79 ida=0x98854f
func TestBuddyModeOnlyArmsV79(t *testing.T) {
	cases := map[byte]func(byte) []byte{
		11: func(b byte) []byte { return NewListFull(b).Encode(nil, nil)(nil) },
		12: func(b byte) []byte { return NewOtherListFull(b).Encode(nil, nil)(nil) },
		14: func(b byte) []byte { return NewCannotBuddyGm(b).Encode(nil, nil)(nil) },
		15: func(b byte) []byte { return NewCharacterNotFound(b).Encode(nil, nil)(nil) },
	}
	for mode, enc := range cases {
		if got := enc(mode); !bytes.Equal(got, []byte{mode}) {
			t.Errorf("v79 mode-only arm mode %d: got % x want %02x", mode, got, mode)
		}
	}
}

// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError version=gms_v79 ida=0x98854f
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError2 version=gms_v79 ida=0x98854f
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError3 version=gms_v79 ida=0x98854f
// packet-audit:verify packet=buddy/clientbound/BuddyUnknownError4 version=gms_v79 ida=0x98854f
func TestBuddyUnknownErrorArmsV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	cases := map[byte][]byte{
		16: NewUnknownError(16).Encode(nil, ctx)(nil),
		17: NewUnknownError2(17).Encode(nil, ctx)(nil),
		19: NewUnknownError3(19).Encode(nil, ctx)(nil),
		22: NewUnknownError4(22).Encode(nil, ctx)(nil),
	}
	for mode, got := range cases {
		if want := []byte{mode, 0x00}; !bytes.Equal(got, want) {
			t.Errorf("v79 UnknownError mode %d: got % x want % x", mode, got, want)
		}
	}
}

// packet-audit:verify packet=buddy/clientbound/BuddyUpdate version=gms_v79 ida=0x98854f
// packet-audit:verify packet=buddy/clientbound/BuddyListUpdate version=gms_v79 ida=0x98854f
// packet-audit:verify packet=buddy/clientbound/BuddyInvite version=gms_v79 ida=0x98854f
// packet-audit:verify packet=buddy/clientbound/BuddyChannelChange version=gms_v79 ida=0x98854f
// packet-audit:verify packet=buddy/clientbound/BuddyCapacityUpdate version=gms_v79 ida=0x98854f
func TestBuddyDataArmsV79(t *testing.T) {
	v79 := pt.CreateContext("GMS", 79, 1)
	v83 := pt.CreateContext("GMS", 83, 1)

	// Update (mode 8): charId + GW_Friend(39) + inShop.
	up := NewBuddyUpdate(8, 1000, "TestPlayer", "Default Group", 1, false)
	if a, b := up.Encode(nil, v79)(nil), up.Encode(nil, v83)(nil); !bytes.Equal(a, b) {
		t.Errorf("Update v79 != v83\n v79: % x\n v83: % x", a, b)
	}
	// ListUpdate (mode 7): count + n*GW_Friend(39) + n*inShop(4).
	lu := NewBuddyListUpdate(7, []BuddyEntry{
		{CharacterId: 1000, Name: "Player1", ChannelId: 1, Group: "Default Group", InShop: false},
		{CharacterId: 2000, Name: "Player2", ChannelId: 2, Group: "Friends", InShop: true},
	})
	if a, b := lu.Encode(nil, v79)(nil), lu.Encode(nil, v83)(nil); !bytes.Equal(a, b) {
		t.Errorf("ListUpdate v79 != v83\n v79: % x\n v83: % x", a, b)
	}
	// Invite (mode 9): originatorId + name + GW_Friend(39) + inShop; NO jobId/level on v79 (<87).
	inv := NewBuddyInvite(9, 1000, 2000, "TestPlayer", 510, 120)
	if a, b := inv.Encode(nil, v79)(nil), inv.Encode(nil, v83)(nil); !bytes.Equal(a, b) {
		t.Errorf("Invite v79 != v83\n v79: % x\n v83: % x", a, b)
	}
	if got := inv.Encode(nil, v79)(nil); len(got) != 57 {
		t.Errorf("Invite v79 length: got %d want 57 (no jobId/level for <87)", len(got))
	}
	// ChannelChange (mode 20): charId + inShop(1) + channel(4).
	cc := NewBuddyChannelChange(20, 1000, 3)
	if a, b := cc.Encode(nil, v79)(nil), cc.Encode(nil, v83)(nil); !bytes.Equal(a, b) {
		t.Errorf("ChannelChange v79 != v83\n v79: % x\n v83: % x", a, b)
	}
	// CapacityUpdate (mode 21): capacity byte.
	cu := NewBuddyCapacityUpdate(21, 50)
	if got := cu.Encode(nil, v79)(nil); !bytes.Equal(got, []byte{21, 50}) {
		t.Errorf("CapacityUpdate v79: got % x want 15 32", got)
	}
}
