package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Wire-fix proof for the GW_Friend record-size divergence (task-113 v61).
//
// IDA-verified (GMS_v61.1_U_DEVM.exe, port 13338): GW_Friend::Decode @0x4b54d8
// is DecodeBuffer(this, 22) — a 22-byte record — whereas v72+ GW_Friend::Decode
// is DecodeBuffer(39). The 17-byte delta is the trailing FriendGroup field,
// which buddy groups introduced after v61. CWvsContext::OnFriendResult case
// 0x14 @0x858a20 reads the stored channelId at record offset 18 (`22*Index +
// 18`), fixing the v61 layout as FriendId(4) + FriendName(13) + Flag(1) +
// ChannelId(4) = 22, with FriendGroup absent.
//
// Atlas's model.Buddy.Encode previously always wrote the 17-byte group,
// producing a 39-byte record for every version — WRONG for v61. The
// BuddyHasFriendGroup gate drops it for GMS < 72 only; v72+/JMS unchanged.

// TestBuddyGroupGateV61Update proves the exact v61 Update (mode 8) wire bytes:
// mode + characterId + 22-byte GW_Friend record (no group) + inShop.
func TestBuddyGroupGateV61Update(t *testing.T) {
	v61 := pt.CreateContext("GMS", 61, 1)
	got := NewBuddyUpdate(8, 1000, "TestPlayer", "Default Group", 1, false).Encode(nil, v61)(nil)
	want := []byte{
		0x08,                   // mode (OnFriendResult switch case 8)
		0xE8, 0x03, 0x00, 0x00, // characterId 1000 (Decode4 in UpdateFriend @0x859532)
		// --- GW_Friend record, DecodeBuffer(22) @0x4b54e4 ---
		0xE8, 0x03, 0x00, 0x00, // FriendId 1000
		0x54, 0x65, 0x73, 0x74, 0x50, 0x6C, 0x61, 0x79, 0x65, 0x72, 0x00, 0x00, 0x00, // "TestPlayer" padded to 13
		0x00,                   // Flag
		0x01, 0x00, 0x00, 0x00, // ChannelId 1 (record offset 18, per case 0x14 @0x858a20)
		// FriendGroup ABSENT in v61 (would be 17 bytes in v72+)
		0x00, // inShop false (Decode1 @0x85956d)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("v61 Update record must be 22-byte (no group):\n got % x (len %d)\nwant % x (len %d)", got, len(got), want, len(want))
	}
}

// TestBuddyGroupGateVersionDelta proves the 17-byte-per-record delta between
// v61 and v83 for every model.Buddy-bearing arm, and that v72/v83 are byte-equal
// (v72+ codec path unchanged by the gate).
func TestBuddyGroupGateVersionDelta(t *testing.T) {
	v61 := pt.CreateContext("GMS", 61, 1)
	v72 := pt.CreateContext("GMS", 72, 1)
	v83 := pt.CreateContext("GMS", 83, 1)

	// Update: 1 record → delta 17.
	up := NewBuddyUpdate(8, 1000, "TestPlayer", "Default Group", 1, false)
	if l61, l83 := len(up.Encode(nil, v61)(nil)), len(up.Encode(nil, v83)(nil)); l83-l61 != 17 {
		t.Errorf("Update delta: v83 %d - v61 %d = %d, want 17", l83, l61, l83-l61)
	}
	if a, b := up.Encode(nil, v72)(nil), up.Encode(nil, v83)(nil); !bytes.Equal(a, b) {
		t.Errorf("Update v72 != v83 (gate must not touch v72+)")
	}

	// Invite: 1 record → delta 17. v61 = 40, v83 = 57.
	inv := NewBuddyInvite(9, 1000, 2000, "TestPlayer", 510, 120)
	if l61, l83 := len(inv.Encode(nil, v61)(nil)), len(inv.Encode(nil, v83)(nil)); l83-l61 != 17 {
		t.Errorf("Invite delta: v83 %d - v61 %d = %d, want 17", l83, l61, l83-l61)
	}
	if a, b := inv.Encode(nil, v72)(nil), inv.Encode(nil, v83)(nil); !bytes.Equal(a, b) {
		t.Errorf("Invite v72 != v83 (gate must not touch v72+)")
	}

	// ListUpdate: 2 records → delta 34.
	lu := NewBuddyListUpdate(7, []BuddyEntry{
		{CharacterId: 1000, Name: "Player1", ChannelId: 1, Group: "Default Group", InShop: false},
		{CharacterId: 2000, Name: "Player2", ChannelId: 2, Group: "Friends", InShop: true},
	})
	if l61, l83 := len(lu.Encode(nil, v61)(nil)), len(lu.Encode(nil, v83)(nil)); l83-l61 != 34 {
		t.Errorf("ListUpdate delta: v83 %d - v61 %d = %d, want 34", l83, l61, l83-l61)
	}
	if a, b := lu.Encode(nil, v72)(nil), lu.Encode(nil, v83)(nil); !bytes.Equal(a, b) {
		t.Errorf("ListUpdate v72 != v83 (gate must not touch v72+)")
	}
}
