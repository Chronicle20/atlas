package interaction

import (
	"testing"

	"github.com/Chronicle20/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas-packet/model"
	pt "github.com/Chronicle20/atlas-packet/test"
)

func testAvatar() model.Avatar {
	equip := map[slot.Position]uint32{5: 1040002, 6: 1060002, 7: 1072001}
	masked := map[slot.Position]uint32{}
	pets := map[int8]uint32{}
	return model.NewAvatar(0, 1, 20000, false, 30000, equip, masked, pets)
}

func TestBaseVisitorRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewBaseVisitor(1, testAvatar(), "TestPlayer")
			output := Visitor{visitorType: BaseVisitorType}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Slot() != 1 {
				t.Errorf("slot: got %v, want 1", output.Slot())
			}
			if output.Name() != "TestPlayer" {
				t.Errorf("name: got %q, want %q", output.Name(), "TestPlayer")
			}
		})
	}
}

func TestGameVisitorRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			record := GameRecord{Unknown: 0, Wins: 10, Ties: 2, Losses: 5, Points: 100}
			input := NewGameVisitor(0, testAvatar(), "Gamer", record)
			output := Visitor{visitorType: GameVisitorType}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Name() != "Gamer" {
				t.Errorf("name: got %q, want %q", output.Name(), "Gamer")
			}
			if output.Record().Wins != 10 {
				t.Errorf("wins: got %v, want 10", output.Record().Wins)
			}
		})
	}
}

func TestMerchantVisitorRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewMerchantVisitor(5030000, "MyShop")
			output := Visitor{visitorType: MerchantVisitorType}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.ItemId() != 5030000 {
				t.Errorf("itemId: got %v, want 5030000", output.ItemId())
			}
			if output.MerchantName() != "MyShop" {
				t.Errorf("merchantName: got %q, want %q", output.MerchantName(), "MyShop")
			}
		})
	}
}
