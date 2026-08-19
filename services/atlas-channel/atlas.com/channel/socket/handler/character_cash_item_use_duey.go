package handler

import (
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// dueyCouponSagaCreateFunc is a test seam for saga creation.
var dueyCouponSagaCreateFunc = func(l logrus.FieldLogger, ctx context.Context, s saga.Saga) error {
	return saga.NewProcessor(l, ctx).Create(s)
}

// dueyCouponEnabled reports whether this tenant's client can send
// CASH_ITEM_USE for classification 533 (Quick Delivery Ticket) at all.
//
// GMS half: derived the same way remoteMerchantEnabled derives its GMS gate
// (task-221 design §1.3) — task-241 design §9.5 places the Quick Delivery
// Ticket send path from v72 onward.
//
// JMS half: JMS v185 is enabled. Two facts settle it (task-241 task-22
// controller addendum, IDA-verified, MapleStory_dump_SCY.exe session
// 05eb9c27):
//  1. get_cashslot_item_type @0x49a1ee contains `case 533: return 32;` — JMS
//     routes classification 533 to cash-slot type 32.
//  2. CWvsContext::SendConsumeCashItemUseRequest @0xaef2f5 dispatches through
//     a jump table at @0xaef3a8, and IDA's own default-arm annotation on the
//     bound check at @0xaef3a2 reads verbatim: "ja def_AEF3A8; jumptable
//     00AEF3A8 default case, cases 34,35,37,40,45,46,53,54,62,65-68". 32 is
//     not in that default set, so type 32 has a real, non-default arm — the
//     JMS client actually sends this op.
//
// The server-side response path is already wired for JMS too:
// docs/packets/dispatchers/parcel.yaml:67 gives jms_v185: 10 for OPEN and
// :85 gives jms_v185: 27 for OPEN_QUICK, so PARCEL[OPEN_QUICK] renders on
// jms_v185.
func dueyCouponEnabled(t tenant.Model) bool {
	return (t.IsRegion("GMS") && t.MajorAtLeast(72)) ||
		(t.IsRegion("JMS") && t.MajorAtLeast(185))
}

// handleDueyCouponUse implements classification 533 (Quick Delivery Ticket,
// item 5330000): it announces Duey's quick-send-only dialog
// (PARCEL[OPEN_QUICK]) without visiting the NPC.
//
// Dispatch is classification-first, not cash-slot-type-first, for the same
// reason character_cash_item_use.go:784-786 already documents for the
// megaphone/remote-merchant branches: the cash-slot type byte collides
// across features.
//
// Unlike handleRemoteMerchantUse, there is no cash-slot-type rejection check
// here, and that is deliberate — do not "restore" one by copying
// remoteMerchantCashSlotType. it is not decoded from the wire; it is
// computed one call up the stack, at character_cash_item_use.go:67, by
// GetCashSlotItemType(t)(itemId) — the very region-aware function this task
// updated for classification 533. Comparing it against a second copy of the
// same table can never fail, so the check would be vacuous.
//
// This handler consumes nothing. The Quick Delivery Ticket is destroyed by
// the parcel_send saga when the player actually sends a parcel (task-241
// FR-26), and the client itself pre-checks CWvsContext::IsExist(5330000)
// before letting the player reach that send. There is no sub-body to
// decode; CASH_ITEM_USE's common header (source slot, item id) is all this
// operation carries.
func handleDueyCouponUse(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, t tenant.Model, itemId item.Id, source slot.Position, it CashSlotItemType) {
	return func(s session.Model, t tenant.Model, itemId item.Id, source slot.Position, it CashSlotItemType) {
		enableActions := func() {
			_ = session.EnableActions(l)(ctx)(wp)(s)
		}

		if !dueyCouponEnabled(t) {
			l.Warnf("Character [%d] attempted quick delivery ticket [%d] on unsupported version [region %s major %d]; ignoring without consuming.", s.CharacterId(), itemId, t.Region(), t.MajorVersion())
			enableActions()
			return
		}

		f := s.Field()
		now := time.Now()
		transactionId := uuid.New()

		sg := saga.Saga{
			TransactionId: transactionId,
			SagaType:      saga.InventoryTransaction,
			InitiatedBy:   "CASH_ITEM_USE",
			Steps: []saga.Step{
				{
					StepId: "show_parcel_quick",
					Status: saga.Pending,
					Action: saga.ShowParcel,
					Payload: saga.ShowParcelPayload{
						CharacterId: s.CharacterId(),
						NpcId:       0,
						WorldId:     f.WorldId(),
						ChannelId:   f.ChannelId(),
						Quick:       true,
					},
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		}

		if err := dueyCouponSagaCreateFunc(l, ctx, sg); err != nil {
			l.WithError(err).Errorf("Character [%d] quick delivery ticket [%d]: unable to create saga.", s.CharacterId(), itemId)
			enableActions()
			return
		}

		l.WithFields(logrus.Fields{
			"character_id":   s.CharacterId(),
			"item_id":        uint32(itemId),
			"cash_slot_type": uint32(it),
			"source_slot":    int16(source),
			"transaction_id": transactionId.String(),
		}).Info("Quick delivery dialog open requested.")
	}
}
