package handler

import (
	"bytes"
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	cashcb "github.com/Chronicle20/atlas/libs/atlas-packet/cash/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestCashShopEnableEquipSlotExtSuccessBodyEncodes pins the byte layout of
// the ENABLE_EQUIP_SLOT_EXT_SUCCESS body: mode(1) + slotIndex:uint16 LE(2) +
// days:uint16 LE(2). slotIndex here is 0 -- the WIRE value the channel's
// handleStatusEventEquipSlotIncreased (kafka/consumer/cashshop/consumer.go)
// must pass, never the Atlas canonical -59 the event body carries -- so a
// regression that starts forwarding the event's canonical SlotIndex
// straight through would silently encode 65477 (the unsigned view of -59)
// here instead of 0, and this test would catch it.
func TestCashShopEnableEquipSlotExtSuccessBodyEncodes(t *testing.T) {
	l := logrus.New()
	ten := mustTenant(t, "GMS", 95, 1)
	ctx := tenant.WithContext(context.Background(), ten)
	options := map[string]interface{}{
		"operations": map[string]interface{}{
			cashcb.CashShopOperationEnableEquipSlotExtSuccess: float64(117),
		},
	}

	body := cashcb.CashShopEnableEquipSlotExtSuccessBody(0, 30)(l, ctx)(options)

	want := cashcb.NewEnableEquipSlotExtSuccess(117, 0, 30).Encode(l, ctx)(options)

	if !bytes.Equal(body, want) {
		t.Fatalf("body = %#v, want %#v", body, want)
	}
	if body[0] != 117 {
		t.Errorf("body[0] = %d, want 117 (ENABLE_EQUIP_SLOT_EXT_SUCCESS)", body[0])
	}

	wantExact := []byte{
		117,  // mode: ENABLE_EQUIP_SLOT_EXT_SUCCESS
		0, 0, // slotIndex = 0 (wire value; never the -59 canonical position)
		30, 0, // days = 30, uint16 LE
	}
	if !bytes.Equal(body, wantExact) {
		t.Fatalf("body = %#v, want %#v", body, wantExact)
	}
}
