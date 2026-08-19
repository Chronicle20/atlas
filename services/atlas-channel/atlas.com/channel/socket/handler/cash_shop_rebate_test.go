package handler

import (
	"bytes"
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
)

// TestCashShopRebateDoneBodyEncodes pins the REBATE_SUCCESS arm's wire shape:
// mode(1) + sn:int64 little-endian(8) + amount:int32 little-endian(4)
// (RebateDone.Encode, shop_operation_result_gift.go), and its mode byte, 150
// (CashShopOperationRebateDone == "REBATE_SUCCESS", template_gms_95_1.json).
// The expected bytes are hand-computed from the encoder rather than inferred
// from the Go parameter widths, per the brief.
func TestCashShopRebateDoneBodyEncodes(t *testing.T) {
	l := logrus.New()
	ctx := context.Background()
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			cashcb.CashShopOperationRebateDone: float64(150),
		},
	}

	t.Run("refund", func(t *testing.T) {
		const sn = int64(900001)
		const amount = int32(1200)

		body := cashcb.CashShopRebateDoneBody(sn, amount)(l, ctx)(options)

		want := []byte{
			150,                            // mode: REBATE_SUCCESS
			0xa1, 0xbb, 0xd, 0, 0, 0, 0, 0, // sn = 900001, int64 LE
			0xb0, 0x4, 0, 0, // amount = 1200, int32 LE
		}
		if !bytes.Equal(body, want) {
			t.Fatalf("body = %#v, want %#v", body, want)
		}
		if body[0] != 150 {
			t.Errorf("body[0] = %d, want 150 (REBATE_SUCCESS)", body[0])
		}
	})
}

// TestCashShopRebateFailedBodyMode pins the REBATE_FAILED arm's mode byte,
// 151 (CashShopOperationRebateFailed == "REBATE_FAILED", template_gms_95_1.json)
// -- distinct from the success arm's 150 above -- so a swap of the two
// constants fails this suite (controller correction C5: task 8 shipped
// exactly that swap past a suite with no counterpart test).
func TestCashShopRebateFailedBodyMode(t *testing.T) {
	l := logrus.New()
	ctx := context.Background()
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			cashcb.CashShopOperationRebateFailed: float64(151),
		},
		"errors": map[string]interface{}{
			"INVALID_BIRTHDAY": float64(1),
		},
	}

	body := cashcb.CashShopRebateFailedBody("INVALID_BIRTHDAY")(l, ctx)(options)

	if len(body) == 0 {
		t.Fatal("body is empty")
	}
	if body[0] != 151 {
		t.Errorf("body[0] = %d, want 151 (REBATE_FAILED)", body[0])
	}
	if body[0] == 150 {
		t.Fatalf("body[0] = %d matches the SUCCESS mode byte -- failure and success arms must not share a mode byte", body[0])
	}
}
