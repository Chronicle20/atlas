package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"

	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
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

// buyOtherPackagePacket builds a full BUY_OTHER_PACKAGE wire packet: the
// dispatcher's own mode byte (ShopOperation, cash_shop_operation.go:52-55)
// followed by ShopOperationBuyOtherPackage's body (spw string, serialNumber
// uint32, name string, message string --
// libs/atlas-packet/cash/serverbound/shop_operation_buy_other_package.go:44-53).
// Mirrors buyNameChangePacket/buyWorldTransferPacket
// (cash_shop_operation_imprint_test.go:40-61).
func buyOtherPackagePacket(t *testing.T, op byte, spw string, serialNumber uint32, name, message string) *request.Reader {
	t.Helper()
	w := response.NewWriter(logrus.New())
	w.WriteByte(op)
	w.WriteAsciiString(spw)
	w.WriteInt(serialNumber)
	w.WriteAsciiString(name)
	w.WriteAsciiString(message)
	req := request.Request(w.Bytes())
	reader := request.NewRequestReader(&req, 0)
	return &reader
}

// newBuyOtherPackageTestServer stands in for atlas-account (secondary
// credential resolution) and atlas-character (recipient lookup), routed by
// path. The account response carries no PIC/birthDate, so
// verifySecondaryCredential (cash_shop_credential.go) passes unconditionally
// (credentialMatches: an unset credential of the applicable kind always
// passes). The character response is an empty JSON:API list -- GetByName
// (character/processor.go:236-238) resolves that to atlasmodel.ErrEmptySlice,
// the same "unknown recipient" rejection TestGiftRejectionReason
// (cash_shop_gift_test.go) pins to the "INCORRECT_NAME" errors-table key --
// which is exactly the observable effect asserted below: an announced
// GIFT_PACKAGE_FAILED write on cashcb.CashShopOperationWriter. That write can
// only happen if the dispatcher actually reached handleBuyOtherPackage
// (cash_shop_operation.go:201-206), so it is a direct probe of the dispatch
// arm, not of the isCashShopOperation helper alone.
func newBuyOtherPackageTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "accounts/"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(jsonAPIAttrs("accounts", "1", map[string]any{"name": "Sender"}))
		default:
			// character lookup by name: empty JSON:API list -> unknown recipient.
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[]}`))
		}
	}))
}

// TestBuyOtherPackageIsDispatched is the acceptance criterion for this task
// turned into a test: BUY_OTHER_PACKAGE (CashShopOperationBuyOtherPackage,
// cash_shop_operation.go:40) was declared but referenced nowhere else
// (unrouted). It must now dispatch on its configured op byte and drive
// CashShopOperationHandleFunc -- the real dispatcher entry point -- end to
// end, not just the isCashShopOperation helper: deleting the dispatch arm at
// cash_shop_operation.go:201-206 must turn this test RED.
func TestBuyOtherPackageIsDispatched(t *testing.T) {
	const characterId = uint32(54321)
	const op = byte(33)
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			CashShopOperationBuyOtherPackage: float64(op),
		},
	}

	srv := newBuyOtherPackageTestServer(t)
	defer srv.Close()
	t.Setenv("ACCOUNTS_SERVICE_URL", srv.URL+"/")
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")

	s, ctx, cleanup := newCashItemUseTestSession(t, characterId)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	pkt := buyOtherPackagePacket(t, op, "", 5990000, "UnknownRecipient", "hello")
	CashShopOperationHandleFunc(logrus.New(), ctx, rec.producer())(s, pkt, options)

	if rec.calls != 1 {
		t.Fatalf("announced packets = %d, want 1 (BUY_OTHER_PACKAGE must reach handleBuyOtherPackage and answer GIFT_PACKAGE_FAILED)", rec.calls)
	}
	if rec.lastName != cashcb.CashShopOperationWriter {
		t.Fatalf("announced writer = %q, want %q", rec.lastName, cashcb.CashShopOperationWriter)
	}
}
