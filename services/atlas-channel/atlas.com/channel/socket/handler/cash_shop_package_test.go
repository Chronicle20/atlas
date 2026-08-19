package handler

import (
	"bytes"
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestCashShopPackageResultBodies pins the two success arms handleBuyPackage
// (BUY_PACKAGE_SUCCESS, mode 154) and handleBuyOtherPackage
// (GIFT_PACKAGE_SUCCESS, mode 156) answer on -- derivation.md D3b (§5).
// Every subtest asserts the FULL expected byte slice, not just the mode
// byte, and cross-checks the first byte against the OTHER subtest's mode so
// a 154/156 swap (the standing mutation-testing bar on this branch) fails in
// both directions, not just one.
func TestCashShopPackageResultBodies(t *testing.T) {
	l := logrus.New()
	ten := mustTenant(t, "GMS", 95, 1)
	ctx := tenant.WithContext(context.Background(), ten)
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			cashcb.CashShopOperationBuyPackageDone:  float64(154),
			cashcb.CashShopOperationGiftPackageDone: float64(156),
		},
	}

	t.Run("buy package", func(t *testing.T) {
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
		body := cashcb.CashShopBuyPackageDoneBody([]cashcb.CashInventoryItem{item}, 0)(l, ctx)(options)

		// Hand-computed from BuyPackageDone.Encode + CashInventoryItem.EncodeBytes
		// (shop_operation_result_gift.go / shop_inventory.go): mode(1) +
		// itemCount:byte(1) + one 55-byte CashInventoryItem row
		// (CashId:int64 LE(8) + AccountId:int LE(4) + CharacterId:int LE(4) +
		// TemplateId:int LE(4) + CommodityId:int LE(4) + Quantity:int16 LE(2) +
		// GiftFrom:padded[13] + Expiration:int64 LE(8) + two trailing zero
		// ints LE(4+4)) + trailingCount:uint16 LE(2).
		want := []byte{
			154, // mode: BUY_PACKAGE_SUCCESS
			1,   // item count
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
			0, 0, // trailingCount = 0, uint16 LE
		}

		if !bytes.Equal(body, want) {
			t.Fatalf("body = %#v, want %#v", body, want)
		}
		if body[0] != 154 {
			t.Errorf("body[0] = %d, want 154 (BUY_PACKAGE_SUCCESS)", body[0])
		}
		if body[0] == 156 {
			t.Fatalf("body[0] = %d matches GIFT_PACKAGE_SUCCESS's mode -- BUY_PACKAGE and BUY_OTHER_PACKAGE must not share a mode byte", body[0])
		}
	})

	t.Run("gift package", func(t *testing.T) {
		body := cashcb.CashShopGiftPackageDoneBody("Recipient", 9100000, 0, 0, 3000)(l, ctx)(options)

		// Hand-computed from GiftPackageDone.Encode (shop_operation_result_gift.go):
		// mode(1) + recipientName:asciiString [uint16 LE length(9) + 9 ASCII
		// bytes] + packageId:int32 LE(9100000) + unused1:uint16 LE(0) +
		// unused2:uint16 LE(0) + nxCashSpent:int32 LE(3000) [GMS-only,
		// giftHasNxCashSpent].
		want := []byte{
			156,  // mode: GIFT_PACKAGE_SUCCESS
			9, 0, // "Recipient" length prefix, uint16 LE
			'R', 'e', 'c', 'i', 'p', 'i', 'e', 'n', 't',
			224, 218, 138, 0, // packageId = 9100000, int32 LE
			0, 0, // unused1 = 0, uint16 LE
			0, 0, // unused2 = 0, uint16 LE
			184, 11, 0, 0, // nxCashSpent = 3000, int32 LE
		}

		if !bytes.Equal(body, want) {
			t.Fatalf("body = %#v, want %#v", body, want)
		}
		if body[0] != 156 {
			t.Errorf("body[0] = %d, want 156 (GIFT_PACKAGE_SUCCESS)", body[0])
		}
		if body[0] == 154 {
			t.Fatalf("body[0] = %d matches BUY_PACKAGE_SUCCESS's mode -- BUY_PACKAGE and BUY_OTHER_PACKAGE must not share a mode byte", body[0])
		}
	})
}

// TestBuyOtherPackageIsDispatched is the acceptance criterion for this task
// turned into a test: BUY_OTHER_PACKAGE (CashShopOperationBuyOtherPackage,
// cash_shop_operation.go:40) was declared but referenced nowhere else
// (unrouted). It must now dispatch on its configured op byte and NOT on a
// neighbouring one.
func TestBuyOtherPackageIsDispatched(t *testing.T) {
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			"BUY_OTHER_PACKAGE": float64(33),
		},
	}

	if got := isCashShopOperation(logrus.New())(options, 33, CashShopOperationBuyOtherPackage); !got {
		t.Fatalf("isCashShopOperation(options, 33, %q) = %v, want true", CashShopOperationBuyOtherPackage, got)
	}
	if got := isCashShopOperation(logrus.New())(options, 32, CashShopOperationBuyOtherPackage); got {
		t.Fatalf("isCashShopOperation(options, 32, %q) = %v, want false", CashShopOperationBuyOtherPackage, got)
	}
}
