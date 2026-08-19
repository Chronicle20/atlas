package handler

import (
	"bytes"
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestCashShopRingResultBodies pins the two success arms handleBuyCouple
// (COUPLE_SUCCESS) and handleBuyFriendship (FRIENDSHIP_SUCCESS) answer on --
// the same byte-for-byte assertion style TestCashShopPackageResultBodies
// uses, so a mode-byte swap between the two fails in both directions.
func TestCashShopRingResultBodies(t *testing.T) {
	l := logrus.New()
	ten := mustTenant(t, "GMS", 95, 1)
	ctx := tenant.WithContext(context.Background(), ten)
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			cashcb.CashShopOperationCoupleDone:     float64(152),
			cashcb.CashShopOperationFriendshipDone: float64(162),
		},
	}

	item := cashcb.CashInventoryItem{
		CashId:      1,
		AccountId:   2,
		CharacterId: 3,
		TemplateId:  4,
		CommodityId: 5,
		Quantity:    6,
		GiftFrom:    "",
		Expiration:  7,
	}

	t.Run("couple", func(t *testing.T) {
		body := cashcb.CashShopCoupleDoneBody(item, "Partner", 1112000, 1)(l, ctx)(options)

		// Hand-computed from CoupleDone.Encode + CashInventoryItem.EncodeBytes
		// (shop_operation_result_gift.go / shop_inventory.go): mode(1) + one
		// 55-byte CashInventoryItem row (CashId:int64 LE(8) + AccountId:int
		// LE(4) + CharacterId:int LE(4) + TemplateId:int LE(4) +
		// CommodityId:int LE(4) + Quantity:int16 LE(2) + GiftFrom:padded[13] +
		// Expiration:int64 LE(8) + two trailing zero ints LE(4+4)) +
		// recipientName:EncodeStr("Partner") + itemId:int32 LE(4) +
		// quantity:uint16 LE(2).
		want := []byte{
			152, // mode: COUPLE_SUCCESS
			// CashInventoryItem row:
			1, 0, 0, 0, 0, 0, 0, 0, // CashId = 1, int64 LE
			2, 0, 0, 0, // AccountId = 2, uint32 LE
			3, 0, 0, 0, // CharacterId = 3, uint32 LE
			4, 0, 0, 0, // TemplateId = 4, uint32 LE
			5, 0, 0, 0, // CommodityId = 5, uint32 LE
			6, 0, // Quantity = 6, int16 LE
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // GiftFrom "" padded to 13
			7, 0, 0, 0, 0, 0, 0, 0, // Expiration = 7, int64 LE
			0, 0, 0, 0, // trailing zero int
			0, 0, 0, 0, // trailing zero int
			// recipientName "Partner", length-prefixed ascii string
			7, 0, 'P', 'a', 'r', 't', 'n', 'e', 'r',
			// itemId = 1112000, int32 LE
			192, 247, 16, 0,
			1, 0, // quantity = 1, uint16 LE
		}

		if !bytes.Equal(body, want) {
			t.Fatalf("body = %#v, want %#v", body, want)
		}
		if body[0] != 152 {
			t.Errorf("body[0] = %d, want 152 (COUPLE_SUCCESS)", body[0])
		}
		if body[0] == 162 {
			t.Errorf("body[0] = %d, matches FRIENDSHIP_SUCCESS mode -- couple/friendship mode swap", body[0])
		}
	})

	t.Run("friendship", func(t *testing.T) {
		body := cashcb.CashShopFriendshipDoneBody(item, "Partner", 1112800, 1)(l, ctx)(options)

		want := []byte{
			162, // mode: FRIENDSHIP_SUCCESS
			// CashInventoryItem row:
			1, 0, 0, 0, 0, 0, 0, 0, // CashId = 1, int64 LE
			2, 0, 0, 0, // AccountId = 2, uint32 LE
			3, 0, 0, 0, // CharacterId = 3, uint32 LE
			4, 0, 0, 0, // TemplateId = 4, uint32 LE
			5, 0, 0, 0, // CommodityId = 5, uint32 LE
			6, 0, // Quantity = 6, int16 LE
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // GiftFrom "" padded to 13
			7, 0, 0, 0, 0, 0, 0, 0, // Expiration = 7, int64 LE
			0, 0, 0, 0, // trailing zero int
			0, 0, 0, 0, // trailing zero int
			// recipientName "Partner", length-prefixed ascii string
			7, 0, 'P', 'a', 'r', 't', 'n', 'e', 'r',
			// itemId = 1112800, int32 LE
			224, 250, 16, 0,
			1, 0, // quantity = 1, uint16 LE
		}

		if !bytes.Equal(body, want) {
			t.Fatalf("body = %#v, want %#v", body, want)
		}
		if body[0] != 162 {
			t.Errorf("body[0] = %d, want 162 (FRIENDSHIP_SUCCESS)", body[0])
		}
		if body[0] == 152 {
			t.Errorf("body[0] = %d, matches COUPLE_SUCCESS mode -- couple/friendship mode swap", body[0])
		}
	})
}

// TestRingTypeForOperation pins the pure mapper handleBuyCouple/handleBuyFriendship
// share to select COUPLE vs FRIENDSHIP on RequestRingPurchaseCommandBody.RingType.
func TestRingTypeForOperation(t *testing.T) {
	if got := ringTypeForOperation(CashShopOperationBuyCouple); got != "COUPLE" {
		t.Errorf("ringTypeForOperation(CashShopOperationBuyCouple) = %q, want %q", got, "COUPLE")
	}
	if got := ringTypeForOperation(CashShopOperationBuyFriendship); got != "FRIENDSHIP" {
		t.Errorf("ringTypeForOperation(CashShopOperationBuyFriendship) = %q, want %q", got, "FRIENDSHIP")
	}
}
