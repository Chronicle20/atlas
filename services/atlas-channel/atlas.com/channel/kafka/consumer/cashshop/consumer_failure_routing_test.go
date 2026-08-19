package cashshop

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
)

// TestFailureBodyForOperation pins failureBodyForOperation's routing table:
// each cash-shop arm's ErrorEventBody.Operation value must answer on that
// arm's own *_FAILED mode byte, and an empty or unrecognized operation must
// keep today's exact behavior -- the CashShopInventoryCapacityIncreaseFailedBody
// arm -- byte for byte, so a producer that predates the Operation field is
// unaffected.
func TestFailureBodyForOperation(t *testing.T) {
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			cashcb.CashShopOperationInventoryCapacityIncreaseFailed: float64(110),
			cashcb.CashShopOperationGiftFailed:                      float64(108),
			cashcb.CashShopOperationBuyNormalFailed:                 float64(159),
			cashcb.CashShopOperationRebateFailed:                    float64(151),
			cashcb.CashShopOperationCoupleFailed:                    float64(153),
			cashcb.CashShopOperationFriendshipFailed:                float64(163),
			cashcb.CashShopOperationBuyPackageFailed:                float64(155),
			cashcb.CashShopOperationGiftPackageFailed:               float64(157),
			cashcb.CashShopOperationEnableEquipSlotExtFailed:        float64(118),
		},
		"errors": map[string]interface{}{
			"NOT_ENOUGH_CASH": float64(3),
			"INVENTORY_FULL":  float64(25),
			"unknown_error":   float64(69),
		},
	}

	tests := []struct {
		name       string
		operation  string
		reason     string
		wantMode   byte
		wantErrVal byte
	}{
		{"empty operation keeps today's arm", "", "INVENTORY_FULL", 110, 25},
		{"unknown operation keeps today's arm", "SOMETHING_ELSE", "INVENTORY_FULL", 110, 25},
		{"gift", "GIFT", "NOT_ENOUGH_CASH", 108, 3},
		{"buy normal", "BUY_NORMAL", "NOT_ENOUGH_CASH", 159, 3},
		{"rebate", "REBATE", "unknown_error", 151, 69},
		{"couple", "COUPLE", "NOT_ENOUGH_CASH", 153, 3},
		{"friendship", "FRIENDSHIP", "NOT_ENOUGH_CASH", 163, 3},
		{"buy package", "BUY_PACKAGE", "INVENTORY_FULL", 155, 25},
		{"gift package", "GIFT_PACKAGE", "INVENTORY_FULL", 157, 25},
		{"enable equip slot", "ENABLE_EQUIP_SLOT", "NOT_ENOUGH_CASH", 118, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encode := failureBodyForOperation(tt.operation, tt.reason)
			body := encode(logrus.New(), context.Background())(options)
			if len(body) != 2 {
				t.Fatalf("body = %#v, want 2 bytes (mode, errorCode)", body)
			}
			if body[0] != tt.wantMode {
				t.Fatalf("mode byte = %d, want %d", body[0], tt.wantMode)
			}
			if body[1] != tt.wantErrVal {
				t.Fatalf("errorCode byte = %d, want %d", body[1], tt.wantErrVal)
			}
		})
	}
}
