package drop

import "testing"

// TestModel_OwnType locks the ownType/owner pairing required by
// CDropPool::TryPickUpDrop @0x50463c (GMS v83.1, MapleStory_dump.exe.i64,
// session 754107bf): ownType 0 gates the owner check against the local
// character id, ownType 1 gates it against the local party id, and ownType
// >= 2 (FFA / explosive) skips the owner check entirely. Owner() substitutes
// the party id for the character id whenever the drop is party-owned, so
// OwnType() must switch to 1 in lockstep or every client refuses the pickup
// for the full 15s ownership window (bug-meso-notify-and-party-drop-ownership,
// bug 2).
func TestModel_OwnType(t *testing.T) {
	cases := []struct {
		name         string
		dropType     byte
		ownerId      uint32
		ownerPartyId uint32
		want         byte
	}{
		{name: "character-owned", dropType: 0, ownerId: 1, ownerPartyId: 0, want: 0},
		{name: "party-owned", dropType: 0, ownerId: 1, ownerPartyId: 1000000000, want: 1},
		{name: "type 2 (FFA/explosive) bypasses owner check even when party-owned", dropType: 2, ownerId: 1, ownerPartyId: 1000000000, want: 2},
		{name: "type 3 bypasses owner check even when character-owned", dropType: 3, ownerId: 1, ownerPartyId: 0, want: 3},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewBuilder().
				SetId(1).
				SetType(tc.dropType).
				SetOwner(tc.ownerId, tc.ownerPartyId).
				MustBuild()

			if got := m.OwnType(); got != tc.want {
				t.Errorf("OwnType() = %d, want %d", got, tc.want)
			}
		})
	}
}
