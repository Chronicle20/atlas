package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	testlog "github.com/sirupsen/logrus/hooks/test"
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

// TestCashShopInventoryTrailingShortsGate pins the version gate on the trailing
// slot-counter shorts. IDA v83 CCashShop::OnCashItemResLoadLockerDone@0x4794f6
// reads only 2 trailing shorts (m_nTrunkCount, m_nCharacterSlotCount). v95
// @0x494cb0 reads 2 MORE (m_nBuyCharacterCount, m_nCharacterCount). The empty
// (no-item) body is: byte mode + short count(0) + short storageSlots + short
// characterSlots = 7 bytes for v83; v95 adds 4 bytes (two shorts) = 11.
func TestCashShopInventoryTrailingShortsGate(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	input := NewCashShopInventory(0x58, nil, 4, 3)

	b83 := input.Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	if len(b83) != 7 {
		t.Errorf("v83 length: got %d, want 7 (no trailing buyChar/char shorts)", len(b83))
	}

	b95 := input.Encode(l, pt.CreateContext("GMS", 95, 1))(nil)
	if len(b95) != 11 {
		t.Errorf("v95 length: got %d, want 11 (2 extra trailing shorts)", len(b95))
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
