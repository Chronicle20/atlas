package handler

import (
	"atlas-channel/saga"
	"atlas-channel/session"
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

func TestGetCashSlotItemTypeDueyCoupon(t *testing.T) {
	gms := mustTenant(t, "GMS", 83, 1)
	jms := mustTenant(t, "JMS", 185, 1)

	if got := GetCashSlotItemType(gms)(item.Id(item.QuickDeliveryTicketId)); got != CashSlotItemType(31) {
		t.Errorf("GMS 83: GetCashSlotItemType(%d) = %d, want 31", item.QuickDeliveryTicketId, got)
	}
	if got := GetCashSlotItemType(jms)(item.Id(item.QuickDeliveryTicketId)); got != CashSlotItemType(32) {
		t.Errorf("JMS 185: GetCashSlotItemType(%d) = %d, want 32", item.QuickDeliveryTicketId, got)
	}
}

// installDueyCouponSagaSeam records created sagas instead of producing.
func installDueyCouponSagaSeam(t *testing.T) (*[]saga.Saga, func()) {
	t.Helper()
	var got []saga.Saga
	orig := dueyCouponSagaCreateFunc
	dueyCouponSagaCreateFunc = func(_ logrus.FieldLogger, _ context.Context, s saga.Saga) error {
		got = append(got, s)
		return nil
	}
	return &got, func() { dueyCouponSagaCreateFunc = orig }
}

func TestHandleDueyCouponUse(t *testing.T) {
	const itemId = uint32(item.QuickDeliveryTicketId)
	const charId = uint32(444)
	const srcSlot = int16(3)

	t.Run("emits show_parcel quick", func(t *testing.T) {
		restoreSlot := installCashItemInSlotSeam(t, srcSlot, itemId)
		defer restoreSlot()
		sagas, restoreSaga := installDueyCouponSagaSeam(t)
		defer restoreSaga()

		s, ctx, cleanup := newCashItemUseTestSession(t, charId)
		defer cleanup()

		req := request.Request(cashItemUsePrefix(srcSlot, itemId))
		reader := request.NewRequestReader(&req, 0)
		CharacterCashItemUseHandleFunc(logrus.New(), ctx, nil)(s, &reader, map[string]interface{}{})

		if len(*sagas) != 1 {
			t.Fatalf("sagas created = %d, want 1", len(*sagas))
		}
		sg := (*sagas)[0]
		if len(sg.Steps) != 1 {
			t.Fatalf("steps = %d, want 1", len(sg.Steps))
		}
		if sg.Steps[0].Action != saga.ShowParcel {
			t.Errorf("step 0 action = %q, want show_parcel", sg.Steps[0].Action)
		}
		p, ok := sg.Steps[0].Payload.(saga.ShowParcelPayload)
		if !ok {
			t.Fatalf("step 0 payload = %+v (%T), want ShowParcelPayload", sg.Steps[0].Payload, sg.Steps[0].Payload)
		}
		if !p.Quick {
			t.Error("Quick = false, want true")
		}
		if p.NpcId != 0 {
			t.Errorf("NpcId = %d, want 0", p.NpcId)
		}
		if p.CharacterId != charId {
			t.Errorf("CharacterId = %d, want %d", p.CharacterId, charId)
		}
	})

	t.Run("jms emits show_parcel quick", func(t *testing.T) {
		restoreSlot := installCashItemInSlotSeam(t, srcSlot, itemId)
		defer restoreSlot()
		sagas, restoreSaga := installDueyCouponSagaSeam(t)
		defer restoreSaga()

		s, ctx, cleanup := newCashItemUseTestSessionForVersion(t, charId, "JMS", 185)
		defer cleanup()

		// JMS leads with updateTime (cashsb.UpdateTimeFirst) before the common
		// source/itemId prefix — see character_cash_item_use_test.go's identical
		// v87+/JMS raw-byte prepend.
		raw := append([]byte{0x2A, 0x00, 0x00, 0x00}, cashItemUsePrefix(srcSlot, itemId)...)
		req := request.Request(raw)
		reader := request.NewRequestReader(&req, 0)
		CharacterCashItemUseHandleFunc(logrus.New(), ctx, nil)(s, &reader, map[string]interface{}{})

		if len(*sagas) != 1 {
			t.Fatalf("sagas created = %d, want 1", len(*sagas))
		}
		sg := (*sagas)[0]
		if len(sg.Steps) != 1 {
			t.Fatalf("steps = %d, want 1", len(sg.Steps))
		}
		if sg.Steps[0].Action != saga.ShowParcel {
			t.Errorf("step 0 action = %q, want show_parcel", sg.Steps[0].Action)
		}
		p, ok := sg.Steps[0].Payload.(saga.ShowParcelPayload)
		if !ok {
			t.Fatalf("step 0 payload = %+v (%T), want ShowParcelPayload", sg.Steps[0].Payload, sg.Steps[0].Payload)
		}
		if !p.Quick {
			t.Error("Quick = false, want true")
		}
		if p.NpcId != 0 {
			t.Errorf("NpcId = %d, want 0", p.NpcId)
		}
	})

	t.Run("consumes nothing", func(t *testing.T) {
		restoreSlot := installCashItemInSlotSeam(t, srcSlot, itemId)
		defer restoreSlot()
		sagas, restoreSaga := installDueyCouponSagaSeam(t)
		defer restoreSaga()

		s, ctx, cleanup := newCashItemUseTestSession(t, charId)
		defer cleanup()

		req := request.Request(cashItemUsePrefix(srcSlot, itemId))
		reader := request.NewRequestReader(&req, 0)
		CharacterCashItemUseHandleFunc(logrus.New(), ctx, nil)(s, &reader, map[string]interface{}{})

		if len(*sagas) != 1 {
			t.Fatalf("sagas created = %d, want 1", len(*sagas))
		}
		for _, step := range (*sagas)[0].Steps {
			if step.Action == saga.DestroyAssetFromSlot || step.Action == saga.DestroyAsset {
				t.Errorf("saga destroys the ticket via step %q — FR-26 says it is consumed only by parcel_send", step.StepId)
			}
		}
	})

	t.Run("out of span", func(t *testing.T) {
		restoreSlot := installCashItemInSlotSeam(t, srcSlot, itemId)
		defer restoreSlot()
		sagas, restoreSaga := installDueyCouponSagaSeam(t)
		defer restoreSaga()

		// gms_61 predates the classification-533 send path (design §9.5, same
		// GMS floor as remoteMerchantEnabled).
		s, ctx, cleanup := newRemoteMerchantRejectionTestSession(t, 987, "GMS", 61)
		defer cleanup()

		req := request.Request(cashItemUsePrefix(srcSlot, itemId))
		reader := request.NewRequestReader(&req, 0)
		CharacterCashItemUseHandleFunc(logrus.New(), ctx, remoteMerchantNoopWP)(s, &reader, map[string]interface{}{})

		if len(*sagas) != 0 {
			t.Errorf("sagas created = %d, want 0 on a disabled version", len(*sagas))
		}

		if _, err := session.NewProcessor(logrus.New(), ctx).GetByCharacterId(s.Field().Channel())(987); err != nil {
			t.Errorf("session was removed from the registry, want it left open: %v", err)
		}
	})
}
