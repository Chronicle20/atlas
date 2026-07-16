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
			input := NewPersonalShopRoom(0, visitors, "CoolShop", 16, nil)
			output := Room{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.RoomType() != PersonalShopRoomType {
				t.Errorf("roomType: got %v, want %v", output.RoomType(), PersonalShopRoomType)
			}
			if output.Position() != 0 {
				t.Errorf("position: got %v, want 0", output.Position())
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

// Visitor view: position carries the recipient's actual slot (1..3), and no
// owner-only block exists for personal shops either way.
func TestPersonalShopRoomVisitorSlotRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewPersonalShopRoom(2, nil, "CoolShop", 16, nil)
			output := Room{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Position() != 2 {
				t.Errorf("position: got %v, want 2", output.Position())
			}
		})
	}
}

// Visitor (position != 0) merchant view: no owner ledger block on the wire.
func TestMerchantShopRoomVisitorRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			visitors := []Visitor{
				NewMerchantVisitor(5030000, "HiredMerch"),
			}
			messages := []RoomMessage{
				{Message: "Welcome!", Slot: 0},
			}
			input := NewMerchantShopRoom(1, visitors, messages, "Owner", "MerchShop", 16, 50000, nil)
			output := Room{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.RoomType() != MerchantShopRoomType {
				t.Errorf("roomType: got %v, want %v", output.RoomType(), MerchantShopRoomType)
			}
			if output.Position() != 1 {
				t.Errorf("position: got %v, want 1", output.Position())
			}
			if output.OwnerName() != "Owner" {
				t.Errorf("ownerName: got %q, want %q", output.OwnerName(), "Owner")
			}
			if output.Title() != "MerchShop" {
				t.Errorf("title: got %q, want %q", output.Title(), "MerchShop")
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

// Owner (position 0) merchant view: the open-time/first-time/sale-ledger block
// is present and round-trips.
func TestMerchantShopRoomOwnerLedgerRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			sold := []RoomSoldItem{
				{ItemId: 2000000, Quantity: 3, Price: 500, BuyerName: "Buyer1"},
			}
			input := NewMerchantShopRoom(0, nil, nil, "Owner", "MerchShop", 16, 50000, nil).
				SetOwnerLedger(42, true, sold, 1500)
			output := Room{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Position() != 0 {
				t.Errorf("position: got %v, want 0", output.Position())
			}
			if output.OpenTime() != 42 {
				t.Errorf("openTime: got %v, want 42", output.OpenTime())
			}
			if !output.FirstTime() {
				t.Errorf("firstTime: got false, want true")
			}
			if len(output.SoldItems()) != 1 {
				t.Fatalf("soldItems: got %v, want 1", len(output.SoldItems()))
			}
			if output.SoldItems()[0].BuyerName != "Buyer1" {
				t.Errorf("buyer: got %q, want %q", output.SoldItems()[0].BuyerName, "Buyer1")
			}
			if output.SoldTotal() != 1500 {
				t.Errorf("soldTotal: got %v, want 1500", output.SoldTotal())
			}
		})
	}
}

// TestRoomPositionByteSemantics pins the OnEnterResultBase second header byte
// (v83 @0x65ec6b -> *(this+0xC8)) to the CLIENT's semantics: it is the
// recipient's position in the room — 0 = owner, 1..3 = visitor slot.
// CEntrustedShopDlg's position==0 branch (@0x518a7e) decodes the owner-only
// ledger block and opens the owner management UI (UI_Open gated on
// !position @0x518d3d), so 0 = owner.
func TestRoomPositionByteSemantics(t *testing.T) {
	v := pt.Variants[0]
	ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)

	owner := NewPersonalShopRoom(0, nil, "AB", 16, nil)
	b := owner.Encode(nil, ctx)(nil)
	if b[2] != 0x00 {
		t.Errorf("owner position byte: got %#x, want 0x00 (owner = position 0)", b[2])
	}

	visitor := NewPersonalShopRoom(3, nil, "AB", 16, nil)
	b = visitor.Encode(nil, ctx)(nil)
	if b[2] != 0x03 {
		t.Errorf("visitor position byte: got %#x, want 0x03 (true slot)", b[2])
	}
}
