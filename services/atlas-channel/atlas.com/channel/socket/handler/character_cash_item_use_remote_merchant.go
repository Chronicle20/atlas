package handler

import (
	cashData "atlas-channel/data/cash"
	"atlas-channel/remotemerchant"
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

// cashItemDataFunc is a test seam for the atlas-data cash-item read
// (package-var injection precedent: cashItemInSlotFunc in
// character_cash_item_use.go).
var cashItemDataFunc = func(l logrus.FieldLogger, ctx context.Context, itemId uint32) (cashData.RestModel, error) {
	return cashData.NewProcessor(l, ctx).GetById(itemId)
}

// remoteMerchantSagaCreateFunc is a test seam for saga creation.
var remoteMerchantSagaCreateFunc = func(l logrus.FieldLogger, ctx context.Context, s saga.Saga) error {
	return saga.NewProcessor(l, ctx).Create(s)
}

// remoteMerchantEnabled reports whether this tenant's client can send
// CASH_ITEM_USE for classification 545 at all.
//
// Derived from CWvsContext::SendConsumeCashItemUseRequest's jump table in every
// GMS IDB (task-221 design §1.3): gms_12 registers no CharacterCashItemUseHandle
// at all; gms_48 predates get_cashslot_item_type entirely; gms_61 lists case 37
// in the dispatcher's default arm (@0x832af3), so it computes the type and sends
// nothing. v72 (@0x907472) through v95 (@0x9ee50a) all have a real arm.
//
// JMS maps classification 545 to cash-slot type 36 rather than 37/38
// (get_cashslot_item_type @0x49a1ee) and this task seeds no JMS shops or
// templates, so it stays off — design §7.3 records the bounded follow-up.
func remoteMerchantEnabled(t tenant.Model) bool {
	return t.IsRegion("GMS") && t.MajorAtLeast(72)
}

// remoteMerchantCashSlotType returns the cash-slot type this tenant's client
// computes for a remote store, mirroring GetCashSlotItemType's 545 branch.
func remoteMerchantCashSlotType(t tenant.Model) CashSlotItemType {
	if t.IsRegion("GMS") && t.MajorAtLeast(95) {
		return CashSlotItemType(38)
	}
	return CashSlotItemType(37)
}

// handleRemoteMerchantUse implements classification 545 (remote merchant):
// Miu Miu the Traveling Merchant (5450000) opens NPC 9090000's shop from
// anywhere, and the item is consumed only once the shop is confirmed open.
//
// Dispatch is classification-first, not cash-slot-type-first, for the reason
// character_cash_item_use.go:503-507 already documents: the type byte collides
// (37 is also the wedding-ticket bucket, 59/60 are also triple-megaphone
// buckets). The type is validated here, never used to choose a path.
//
// There is no sub-body to decode. Every version's arm falls straight into the
// dispatcher's shared encode-and-send tail with no Encode* of its own — v83
// @0xa0cda7, v95 @0x9ee50a (design §1.2, OQ-1).
//
// Ownership was already re-validated for every arm in
// CharacterCashItemUseHandleFunc (character_cash_item_use.go:54-58); this arm
// deliberately does not repeat it.
func handleRemoteMerchantUse(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, t tenant.Model, itemId item.Id, source slot.Position, it CashSlotItemType) {
	return func(s session.Model, t tenant.Model, itemId item.Id, source slot.Position, it CashSlotItemType) {
		enableActions := func() {
			_ = session.EnableActions(l)(ctx)(wp)(s)
		}

		// 5451xxx is the Remote Gachapon Ticket. No audited client build emits
		// CASH_ITEM_USE for its cash-slot type — 59/60 sit in every build's
		// default arm (design §1.2, OQ-3) — so reaching this branch means a
		// crafted packet. Never consume.
		if uint32(itemId)/1000 == 5451 {
			l.Warnf("Character [%d] attempted remote gachapon ticket [%d]; no client build emits this op — ignoring without consuming.", s.CharacterId(), itemId)
			enableActions()
			return
		}

		if !remoteMerchantEnabled(t) {
			l.Warnf("Character [%d] attempted remote merchant item [%d] on unsupported version [region %s major %d]; ignoring without consuming.", s.CharacterId(), itemId, t.Region(), t.MajorVersion())
			enableActions()
			return
		}

		if expected := remoteMerchantCashSlotType(t); it != expected {
			l.Warnf("Character [%d] remote merchant item [%d] arrived with cash slot type [%d], expected [%d]. Impossible from a legit client. Rejecting.", s.CharacterId(), itemId, it, expected)
			enableActions()
			return
		}

		ci, err := cashItemDataFunc(l, ctx, uint32(itemId))
		if err != nil {
			l.WithError(err).Errorf("Character [%d] remote merchant item [%d]: unable to read cash item data.", s.CharacterId(), itemId)
			enableActions()
			return
		}
		if ci.Npc == 0 {
			l.Warnf("Character [%d] remote merchant item [%d] resolves to npc 0; no shop to open.", s.CharacterId(), itemId)
			enableActions()
			return
		}

		f := s.Field()
		now := time.Now()
		transactionId := uuid.New()

		// Registry first: a very fast ENTERED must not arrive before the entry
		// that tells the shop consumer to unlock this client.
		remotemerchant.GetRegistry().Put(t, s.CharacterId(), remotemerchant.Entry{
			ItemId: itemId,
			Slot:   source,
			At:     now,
		})

		sg := saga.Saga{
			TransactionId: transactionId,
			SagaType:      saga.RemoteMerchant,
			InitiatedBy:   "CASH_ITEM_USE",
			Steps: []saga.Step{
				{
					StepId: "open_npc_shop",
					Status: saga.Pending,
					Action: saga.OpenNpcShop,
					Payload: saga.OpenNpcShopPayload{
						CharacterId:   s.CharacterId(),
						WorldId:       f.WorldId(),
						ChannelId:     f.ChannelId(),
						NpcTemplateId: ci.Npc,
					},
					CreatedAt: now,
					UpdatedAt: now,
				},
				{
					StepId: "consume_remote_merchant_item",
					Status: saga.Pending,
					Action: saga.DestroyAssetFromSlot,
					Payload: saga.DestroyAssetFromSlotPayload{
						CharacterId:   s.CharacterId(),
						InventoryType: 5, // cash
						Slot:          int16(source),
						Quantity:      1,
						ShowEffect:    false,
						TemplateId:    uint32(itemId),
					},
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		}

		if err := remoteMerchantSagaCreateFunc(l, ctx, sg); err != nil {
			l.WithError(err).Errorf("Character [%d] remote merchant item [%d]: unable to create saga.", s.CharacterId(), itemId)
			remotemerchant.GetRegistry().ClearCharacter(t, s.CharacterId())
			enableActions()
			return
		}

		l.WithFields(logrus.Fields{
			"character_id":    s.CharacterId(),
			"item_id":         uint32(itemId),
			"cash_slot_type":  uint32(it),
			"npc_template_id": ci.Npc,
			"transaction_id":  transactionId.String(),
		}).Info("Remote merchant shop open requested.")
	}
}
