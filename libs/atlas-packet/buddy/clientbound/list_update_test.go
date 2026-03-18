package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestBuddyListUpdateRoundTrip(t *testing.T) {
	buddies := []BuddyEntry{
		{CharacterId: 1000, Name: "Player1", ChannelId: 1, Group: "Default Group", InShop: false},
		{CharacterId: 2000, Name: "Player2", ChannelId: 2, Group: "Friends", InShop: true},
	}
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewBuddyListUpdate(7, buddies)
			output := ListUpdate{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if len(output.Buddies()) != len(input.Buddies()) {
				t.Fatalf("buddy count: got %v, want %v", len(output.Buddies()), len(input.Buddies()))
			}
			for i, ob := range output.Buddies() {
				ib := input.Buddies()[i]
				if ob.CharacterId != ib.CharacterId {
					t.Errorf("buddy[%d].CharacterId: got %v, want %v", i, ob.CharacterId, ib.CharacterId)
				}
				if ob.Name != ib.Name {
					t.Errorf("buddy[%d].Name: got %v, want %v", i, ob.Name, ib.Name)
				}
				if ob.ChannelId != ib.ChannelId {
					t.Errorf("buddy[%d].ChannelId: got %v, want %v", i, ob.ChannelId, ib.ChannelId)
				}
				if ob.Group != ib.Group {
					t.Errorf("buddy[%d].Group: got %v, want %v", i, ob.Group, ib.Group)
				}
				if ob.InShop != ib.InShop {
					t.Errorf("buddy[%d].InShop: got %v, want %v", i, ob.InShop, ib.InShop)
				}
			}
		})
	}
}
