package handler

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"

	atlasmodel "github.com/Chronicle20/atlas/libs/atlas-model/model"
	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestGiftRejectionReason pins the mapper from an edge rejection to its
// errors-table key. Every subtest asserts its own expected string directly
// -- not through a shared loop/helper -- so a wrong-but-uniform mapping (all
// four cases collapsing onto the same key) cannot pass this suite.
func TestGiftRejectionReason(t *testing.T) {
	t.Run("unknown recipient", func(t *testing.T) {
		got := giftRejectionReason(atlasmodel.ErrEmptySlice)
		if got != "INCORRECT_NAME" {
			t.Fatalf("giftRejectionReason(ErrEmptySlice) = %q, want %q", got, "INCORRECT_NAME")
		}
	})

	t.Run("recipient on the sender's own account", func(t *testing.T) {
		got := giftRejectionReason(errGiftOwnAccount)
		if got != "CANNOT_GIFT_TO_OWN_ACCOUNT" {
			t.Fatalf("giftRejectionReason(errGiftOwnAccount) = %q, want %q", got, "CANNOT_GIFT_TO_OWN_ACCOUNT")
		}
	})

	t.Run("credential mismatch", func(t *testing.T) {
		got := giftRejectionReason(ErrCredentialMismatch)
		if got != "INVALID_BIRTHDAY" {
			t.Fatalf("giftRejectionReason(ErrCredentialMismatch) = %q, want %q", got, "INVALID_BIRTHDAY")
		}
	})

	t.Run("anything else", func(t *testing.T) {
		got := giftRejectionReason(errors.New("some other failure"))
		if got != "unknown_error" {
			t.Fatalf("giftRejectionReason(arbitrary error) = %q, want %q", got, "unknown_error")
		}
	})
}

// TestCashShopGiftDoneBodyEncodes pins the GIFT_SUCCESS arm's wire shape:
// mode(1) + recipientName:asciiString + itemId:int32 LE(4) + quantity:uint16
// LE(2) + nxCashSpent:int32 LE(4) (GiftDone.Encode,
// shop_operation_result_gift.go), and its mode byte, 107
// (CashShopOperationGiftDone == "GIFT_SUCCESS", template_gms_95_1.json:4829).
// The expected bytes are computed directly from NewGiftDone's own encoder
// for ("Recipient", 5010000, 1, 1200), not inferred from the Go parameter
// widths.
func TestCashShopGiftDoneBodyEncodes(t *testing.T) {
	l := logrus.New()
	ten := mustTenant(t, "GMS", 95, 1)
	ctx := tenant.WithContext(context.Background(), ten)
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			cashcb.CashShopOperationGiftDone: float64(107),
		},
	}

	body := cashcb.CashShopGiftDoneBody("Recipient", 5010000, 1, 1200)(l, ctx)(options)

	// Hand-computed from GiftDone.Encode (shop_operation_result_gift.go):
	// mode(1) + WriteAsciiString("Recipient") [uint16 LE length(9) + 9
	// ASCII bytes] + itemId int32 LE(5010000) + quantity uint16 LE(1) +
	// nxCashSpent int32 LE(1200) [GMS-only, giftHasNxCashSpent].
	want := []byte{
		107,  // mode: GIFT_SUCCESS
		9, 0, // "Recipient" length prefix, uint16 LE
		'R', 'e', 'c', 'i', 'p', 'i', 'e', 'n', 't',
		0x50, 0x72, 0x4c, 0x00, // itemId = 5010000, int32 LE
		0x01, 0x00, // quantity = 1, uint16 LE
		0xb0, 0x04, 0x00, 0x00, // nxCashSpent = 1200, int32 LE
	}

	if !bytes.Equal(body, want) {
		t.Fatalf("body = %#v, want %#v", body, want)
	}
	if body[0] != 107 {
		t.Errorf("body[0] = %d, want 107 (GIFT_SUCCESS)", body[0])
	}
}

// TestCashShopGiftFailedBodyMode pins the GIFT_FAILED arm's mode byte, 108
// (CashShopOperationGiftFailed == "GIFT_FAILED", template_gms_95_1.json:4830)
// -- distinct from the success arm's 107 above -- so a swap of the two
// constants (task 8's defect class) fails this suite.
func TestCashShopGiftFailedBodyMode(t *testing.T) {
	l := logrus.New()
	ten := mustTenant(t, "GMS", 95, 1)
	ctx := tenant.WithContext(context.Background(), ten)
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			cashcb.CashShopOperationGiftFailed: float64(108),
		},
		"errors": map[string]interface{}{
			"INVALID_BIRTHDAY": float64(1),
		},
	}

	body := cashcb.CashShopGiftFailedBody("INVALID_BIRTHDAY")(l, ctx)(options)

	if len(body) == 0 {
		t.Fatal("body is empty")
	}
	if body[0] != 108 {
		t.Errorf("body[0] = %d, want 108 (GIFT_FAILED)", body[0])
	}
	if body[0] == 107 {
		t.Fatalf("body[0] = %d matches the SUCCESS mode byte -- failure and success arms must not share a mode byte", body[0])
	}
}
