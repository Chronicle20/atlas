package interaction

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// Game rooms (Omok / Match Cards) are intentionally NOT modelled by Room —
// the verified game room-enter blob is clientbound.InteractionMiniGameRoom
// (see interaction/clientbound/interaction_minigame_room_test.go).

func TestPersonalShopRoomRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			visitors := []Visitor{
				NewBaseVisitor(0, testAvatar(), "ShopOwner"),
			}
			input := NewPersonalShopRoom(visitors, "CoolShop", 16, nil)
			output := Room{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.RoomType() != PersonalShopRoomType {
				t.Errorf("roomType: got %v, want %v", output.RoomType(), PersonalShopRoomType)
			}
			if output.Title() != "CoolShop" {
				t.Errorf("title: got %q, want %q", output.Title(), "CoolShop")
			}
			if output.MaxItemCount() != 16 {
				t.Errorf("maxItemCount: got %v, want 16", output.MaxItemCount())
			}
		})
	}
}

func TestMerchantShopRoomRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			visitors := []Visitor{
				NewMerchantVisitor(5030000, "HiredMerch"),
			}
			messages := []RoomMessage{
				{Message: "Welcome!", Slot: 0},
			}
			input := NewMerchantShopRoom(visitors, messages, "Owner", 16, 50000, nil)
			output := Room{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.RoomType() != MerchantShopRoomType {
				t.Errorf("roomType: got %v, want %v", output.RoomType(), MerchantShopRoomType)
			}
			if output.OwnerName() != "Owner" {
				t.Errorf("ownerName: got %q, want %q", output.OwnerName(), "Owner")
			}
			if output.Meso() != 50000 {
				t.Errorf("meso: got %v, want 50000", output.Meso())
			}
			if len(output.Messages()) != 1 {
				t.Fatalf("messages: got %v, want 1", len(output.Messages()))
			}
			if output.Messages()[0].Message != "Welcome!" {
				t.Errorf("message: got %q, want %q", output.Messages()[0].Message, "Welcome!")
			}
		})
	}
}
