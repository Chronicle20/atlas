package handler

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/sirupsen/logrus"

	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
)

// TestCashShopPurchaseRecordDoneBodyEncodesPurchasedFlag pins the wire
// layout CashShopPurchaseRecordDoneBody produces for the GET_PURCHASE_RECORD
// answer: mode byte, then the serial number as a little-endian int32
// (PurchaseRecordDone.Encode's WriteInt32), then the purchased byte.
func TestCashShopPurchaseRecordDoneBodyEncodesPurchasedFlag(t *testing.T) {
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			cashcb.CashShopOperationPurchaseRecordDone: float64(175),
		},
	}

	tests := []struct {
		name      string
		goodsSN   int32
		purchased byte
	}{
		{"purchased", 10000, 1},
		{"not purchased", 10000, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := cashcb.CashShopPurchaseRecordDoneBody(tt.goodsSN, tt.purchased)(logrus.New(), context.Background())(options)
			if len(body) != 6 {
				t.Fatalf("body = %#v, want 6 bytes (mode, int32 goodsSN, purchased)", body)
			}
			if body[0] != 175 {
				t.Fatalf("mode byte = %d, want 175", body[0])
			}
			gotSN := int32(binary.LittleEndian.Uint32(body[1:5]))
			if gotSN != tt.goodsSN {
				t.Fatalf("goodsSN = %d, want %d", gotSN, tt.goodsSN)
			}
			if body[5] != tt.purchased {
				t.Fatalf("purchased byte = %d, want %d", body[5], tt.purchased)
			}
		})
	}
}

// TestPurchaseRecordFlag pins purchaseRecordFlag: any nonzero purchase count
// maps to the wire's "purchased" byte (1), never the literal count.
func TestPurchaseRecordFlag(t *testing.T) {
	tests := []struct {
		count uint32
		want  byte
	}{
		{0, 0},
		{1, 1},
		{7, 1},
	}
	for _, tt := range tests {
		if got := purchaseRecordFlag(tt.count); got != tt.want {
			t.Fatalf("purchaseRecordFlag(%d) = %d, want %d", tt.count, got, tt.want)
		}
	}
}
