package cash

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func testItem() CashInventoryItem {
	return CashInventoryItem{
		CashId:      9000000001,
		AccountId:   100,
		CharacterId: 200,
		TemplateId:  5000000,
		CommodityId: 10001,
		Quantity:    1,
		GiftFrom:    "TestSender",
		Expiration:  150292291200000,
	}
}

func TestCashShopInventoryRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			items := []CashInventoryItem{testItem()}
			input := NewCashShopInventory(0x4D, items, 4, 3)
			output := CashShopInventory{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if len(output.Items()) != 1 {
				t.Fatalf("items: got %d, want 1", len(output.Items()))
			}
			if output.Items()[0].CashId != items[0].CashId {
				t.Errorf("cashId: got %v, want %v", output.Items()[0].CashId, items[0].CashId)
			}
			if output.Items()[0].TemplateId != items[0].TemplateId {
				t.Errorf("templateId: got %v, want %v", output.Items()[0].TemplateId, items[0].TemplateId)
			}
			if output.Items()[0].GiftFrom != items[0].GiftFrom {
				t.Errorf("giftFrom: got %q, want %q", output.Items()[0].GiftFrom, items[0].GiftFrom)
			}
			if output.StorageSlots() != input.StorageSlots() {
				t.Errorf("storageSlots: got %v, want %v", output.StorageSlots(), input.StorageSlots())
			}
			if output.CharacterSlots() != input.CharacterSlots() {
				t.Errorf("characterSlots: got %v, want %v", output.CharacterSlots(), input.CharacterSlots())
			}
		})
	}
}

func TestCashShopInventoryEmptyRoundTrip(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewCashShopInventory(0x4D, nil, 4, 3)
	output := CashShopInventory{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if len(output.Items()) != 0 {
		t.Errorf("items: got %d, want 0", len(output.Items()))
	}
}

func TestCashShopPurchaseSuccessRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCashShopPurchaseSuccess(0x4E, testItem())
			output := CashShopPurchaseSuccess{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.Item().CashId != input.Item().CashId {
				t.Errorf("cashId: got %v, want %v", output.Item().CashId, input.Item().CashId)
			}
			if output.Item().TemplateId != input.Item().TemplateId {
				t.Errorf("templateId: got %v, want %v", output.Item().TemplateId, input.Item().TemplateId)
			}
		})
	}
}

func TestCashShopGiftsRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewCashShopGifts(0x4F)
			output := CashShopGifts{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}
