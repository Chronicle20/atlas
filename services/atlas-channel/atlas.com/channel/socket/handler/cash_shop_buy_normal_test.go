package handler

import (
	"atlas-channel/cashshop"
	messageCashShop "atlas-channel/kafka/message/cashshop"
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// buyNormalPacket builds the v83+ BUY_NORMAL wire body: mode + serialNumber(4).
// ShopOperationBuyNormal carries nothing else on this version
// (shop_operation_buy_normal.go:23-28) -- unlike ShopOperationBuy, there is no
// isPoints/currency to decode.
func buyNormalPacket(t *testing.T, mode byte, serialNumber uint32) *request.Reader {
	t.Helper()
	w := response.NewWriter(logrus.New())
	w.WriteByte(mode)
	w.WriteInt(serialNumber)
	req := request.Request(w.Bytes())
	reader := request.NewRequestReader(&req, 0)
	return &reader
}

// buyNormalOperationsOptions mirrors the config isCashShopOperation reads at
// runtime (cash_shop_operation.go), binding only BUY_NORMAL's mode.
func buyNormalOperationsOptions(mode byte) map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			CashShopOperationBuyNormal: float64(mode),
		},
	}
}

// TestBuyNormalPurchaseCurrency pins the isPoints=false/currency=0 derivation
// documented at the handler's BUY_NORMAL arm and shop_operation_buy_normal.go:23-28
// -- on v83+ the whole client body is a bare serialNumber, so there is
// nothing else to charge with, the identical answer task-227 resolved for
// BUY_NAME_CHANGE. Each case is driven through the same
// cashshop.Processor.RequestPurchase the handler calls -- which applies
// resolvePurchaseCurrency (cashshop/processor.go) internally -- and the
// emitted REQUEST_PURCHASE command's Currency is read back, since
// resolvePurchaseCurrency itself is unexported and lives in a different
// package than this test.
func TestBuyNormalPurchaseCurrency(t *testing.T) {
	tests := []struct {
		name         string
		isPoints     bool
		currency     uint32
		wantCurrency uint32
	}{
		{name: "buy normal sends credit", isPoints: false, currency: 0, wantCurrency: 0},
		{name: "points buy with no currency steers to maple points", isPoints: true, currency: 0, wantCurrency: 2},
		{name: "explicit currency passes through", isPoints: false, currency: 4, wantCurrency: 4},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			captured, restore := installCapturingProducer()
			defer restore()

			const accountId = uint32(9000)
			const characterId = uint32(9001)
			const serial = uint32(5555)

			_, ctx, cleanup := newGachaponTestSession(t, accountId, characterId)
			defer cleanup()

			if err := cashshop.NewProcessor(logrus.New(), ctx).RequestPurchase(characterId, serial, tc.isPoints, tc.currency, 0, uuid.New(), messageCashShop.ErrorOperationBuyNormal); err != nil {
				t.Fatalf("RequestPurchase: %v", err)
			}

			msgs := (*captured)[messageCashShop.EnvCommandTopic]
			if len(msgs) != 1 {
				t.Fatalf("REQUEST_PURCHASE messages emitted = %d, want 1", len(msgs))
			}
			var cmd messageCashShop.Command[messageCashShop.RequestPurchaseCommandBody]
			if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
				t.Fatalf("unmarshal REQUEST_PURCHASE command: %v", err)
			}
			if cmd.Body.Currency != tc.wantCurrency {
				t.Errorf("Body.Currency = %d, want %d", cmd.Body.Currency, tc.wantCurrency)
			}
			if cmd.Body.Operation != messageCashShop.ErrorOperationBuyNormal {
				t.Errorf("Body.Operation = %q, want %q", cmd.Body.Operation, messageCashShop.ErrorOperationBuyNormal)
			}
		})
	}
}

// TestBuyNormalHandleEmitsPurchase drives an actual BUY_NORMAL packet through
// CashShopOperationHandleFunc and asserts the wiring the brief's Step 3
// requires: the decoded serialNumber, isPoints=false/currency=0 (see
// TestBuyNormalPurchaseCurrency), the BUY_NORMAL operation discriminator, and
// a non-nil per-click TransactionId (design §8) all reach the emitted
// REQUEST_PURCHASE command.
func TestBuyNormalHandleEmitsPurchase(t *testing.T) {
	captured, restore := installCapturingProducer()
	defer restore()

	const accountId = uint32(7001)
	const characterId = uint32(7002)
	const serial = uint32(778899)
	const mode = byte(20)

	s, ctx, cleanup := newGachaponTestSession(t, accountId, characterId)
	defer cleanup()

	CashShopOperationHandleFunc(logrus.New(), ctx, nil)(s, buyNormalPacket(t, mode, serial), buyNormalOperationsOptions(mode))

	msgs := (*captured)[messageCashShop.EnvCommandTopic]
	if len(msgs) != 1 {
		t.Fatalf("REQUEST_PURCHASE messages emitted = %d, want 1", len(msgs))
	}
	var cmd messageCashShop.Command[messageCashShop.RequestPurchaseCommandBody]
	if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
		t.Fatalf("unmarshal REQUEST_PURCHASE command: %v", err)
	}

	if cmd.CharacterId != characterId {
		t.Errorf("CharacterId = %d, want %d", cmd.CharacterId, characterId)
	}
	if cmd.Body.SerialNumber != serial {
		t.Errorf("Body.SerialNumber = %d, want %d", cmd.Body.SerialNumber, serial)
	}
	if cmd.Body.Currency != 0 {
		t.Errorf("Body.Currency = %d, want 0", cmd.Body.Currency)
	}
	if cmd.Body.Operation != messageCashShop.ErrorOperationBuyNormal {
		t.Errorf("Body.Operation = %q, want %q", cmd.Body.Operation, messageCashShop.ErrorOperationBuyNormal)
	}
	if cmd.Body.TransactionId == uuid.Nil {
		t.Error("Body.TransactionId = nil UUID, want a per-click minted id")
	}
}

// TestCashShopBuyNormalDoneBodyMode pins the SUCCESS arm this op must
// announce (controller correction C1): BUY_NORMAL_SUCCESS, mode 158 in
// template_gms_95_1.json. The body is round-tripped through BuyNormalDone's
// own Decode rather than a hand-written byte slice, so the assertion tracks
// the real wire shape (mode + int32 count + N packed 8-byte refs) instead of
// a value that could silently drift from the encoder.
func TestCashShopBuyNormalDoneBodyMode(t *testing.T) {
	l := logrus.New()
	ctx := context.Background()
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			cashcb.CashShopOperationBuyNormalDone: float64(158),
		},
	}

	refs := []cashcb.PackedCashItemRef{{Quantity: 1, SlotPos: 0, ItemId: 5990000}}
	body := cashcb.CashShopBuyNormalDoneBody(refs)(l, ctx)(options)

	req := request.Request(body)
	r := request.NewRequestReader(&req, 0)
	decoded := &cashcb.BuyNormalDone{}
	decoded.Decode(l, ctx)(&r, options)

	if decoded.Mode() != 158 {
		t.Errorf("mode byte = %d, want 158 (BUY_NORMAL_SUCCESS)", decoded.Mode())
	}
	if len(decoded.Refs()) != 1 || decoded.Refs()[0] != refs[0] {
		t.Errorf("Refs() = %+v, want %+v", decoded.Refs(), refs)
	}
	if body[0] != 158 {
		t.Errorf("body[0] = %d, want 158", body[0])
	}
}

// TestCashShopBuyNormalFailedBodyMode pins the FAILURE arm this op must
// answer (controller correction C4): BUY_NORMAL_FAILED, mode 159. A prior
// task in this plan shipped a client-wedging bug because only the success
// arm had a test -- this is the failure-arm counterpart.
func TestCashShopBuyNormalFailedBodyMode(t *testing.T) {
	l := logrus.New()
	ctx := context.Background()
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			cashcb.CashShopOperationBuyNormalFailed: float64(159),
		},
		"errors": map[string]interface{}{
			"unknown_error": float64(1),
		},
	}

	body := cashcb.CashShopBuyNormalFailedBody("unknown_error")(l, ctx)(options)

	req := request.Request(body)
	r := request.NewRequestReader(&req, 0)
	decoded := &cashcb.BuyNormalFailed{}
	decoded.Decode(l, ctx)(&r, options)

	if decoded.Mode() != 159 {
		t.Errorf("mode byte = %d, want 159 (BUY_NORMAL_FAILED)", decoded.Mode())
	}
	if body[0] != 159 {
		t.Errorf("body[0] = %d, want 159", body[0])
	}
}
