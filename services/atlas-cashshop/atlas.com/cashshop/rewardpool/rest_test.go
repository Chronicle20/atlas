package rewardpool

import "testing"

// TestTransformReward asserts TransformReward maps every field Model
// declares onto the corresponding RewardRestModel attribute. There is no
// Extract function for this package (rewardpool has no wire→domain
// converter; SelectReward in processor.go builds Model inline from the
// same three fields), so this is a one-directional mapping check rather
// than a round trip.
func TestTransformReward(t *testing.T) {
	m := Model{
		itemId:      2000000,
		quantity:    3,
		commodityId: 5000123,
	}

	rm, err := TransformReward(m)
	if err != nil {
		t.Fatalf("TransformReward: %v", err)
	}

	if rm.ItemId != m.itemId {
		t.Errorf("ItemId mismatch. Expected %v, got %v", m.itemId, rm.ItemId)
	}
	if rm.Quantity != m.quantity {
		t.Errorf("Quantity mismatch. Expected %v, got %v", m.quantity, rm.Quantity)
	}
	if rm.CommodityId != m.commodityId {
		t.Errorf("CommodityId mismatch. Expected %v, got %v", m.commodityId, rm.CommodityId)
	}
}
