package interaction

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestGameRoomRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			visitors := []Visitor{
				NewGameVisitor(0, testAvatar(), "Player1", GameRecord{Wins: 5}),
			}
			input := NewGameRoom(OmokRoomType, 2, visitors, "OmokRoom", 0, false, 0)
			output := Room{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.RoomType() != OmokRoomType {
				t.Errorf("roomType: got %v, want %v", output.RoomType(), OmokRoomType)
			}
			if output.Capacity() != 2 {
				t.Errorf("capacity: got %v, want 2", output.Capacity())
			}
			if len(output.Visitors()) != 1 {
				t.Fatalf("visitors: got %v, want 1", len(output.Visitors()))
			}
			if output.Visitors()[0].Name() != "Player1" {
				t.Errorf("visitor name: got %q, want %q", output.Visitors()[0].Name(), "Player1")
			}
			if output.Title() != "OmokRoom" {
				t.Errorf("title: got %q, want %q", output.Title(), "OmokRoom")
			}
		})
	}
}

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
