package handler

import (
	consumabledata "atlas-channel/data/consumable"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	invsb "github.com/Chronicle20/atlas/libs/atlas-packet/inventory/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// scriptedItemSagaCreateFunc is a test seam for saga creation (precedent:
// remoteMerchantSagaCreateFunc, character_cash_item_use_remote_merchant.go:20-22).
var scriptedItemSagaCreateFunc = func(l logrus.FieldLogger, ctx context.Context, s saga.Saga) error {
	return saga.NewProcessor(l, ctx).Create(s)
}

// scriptedItemDataFunc is a test seam for the consumable data lookup
// (precedent: cashItemDataFunc).
var scriptedItemDataFunc = func(l logrus.FieldLogger, ctx context.Context, itemId uint32) (consumabledata.Model, error) {
	return consumabledata.NewProcessor(l, ctx).GetById(itemId)
}

// evolvingRingUpgradePotionId is the one item outside the 243 family that a v95
// client will send this op for (CWvsContext::SendScriptRunItemRequest's
// itemId/10000 == 243 || itemId == 3994225 gate). It is an Install/Setup item
// and is a documented out-of-scope gap: supporting it needs spec parsing in
// atlas-data's setup reader — which parses no spec node at all today — plus a
// second inventory type on the destroy step. Rejected by name so a play-test
// report explains itself rather than presenting as a mysterious silent drop.
const evolvingRingUpgradePotionId = 3994225

// ScriptedItemHandleFunc handles CWvsContext::SendScriptRunItemRequest — the
// 243xxxx scripted-item route. The item carries its own dialogue, keyed by
// item id and rendered with the avatar named in its WZ spec/npc node.
//
// Ordering is conversation-first: the saga opens the dialogue and only then
// consumes. An item with no authored conversation therefore survives via
// START_ERROR rather than needing a rollback, and there is no pre-flight
// round trip and no TOCTOU window.
//
// Excl-request contract: every rejection path unlocks explicitly; the success
// path does not, because the destroy step's inventory delta is what clears
// the client's m_bExclRequestSent. An explicit unlock as well would
// double-resolve the lock.
func ScriptedItemHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := invsb.ScriptedItem{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		enableActions := func() {
			_ = session.EnableActions(l)(ctx)(wp)(s)
		}

		itemId := item.Id(p.ItemId())

		if uint32(itemId) == evolvingRingUpgradePotionId {
			l.Warnf("Character [%d] used item [%d] (Evolving Ring Upgrade Potion). v95's client whitelists this id alongside the 243 family, but it is an Install/Setup item and is a known out-of-scope gap: atlas-data's setup reader parses no spec node, so its script and npc are unavailable. Not consuming.", s.CharacterId(), itemId)
			enableActions()
			return
		}

		if item.GetClassification(itemId) != item.ClassificationConsumableScriptedItem {
			l.Warnf("Character [%d] attempted scripted item use with non-scripted item [%d]. The client gates this op on itemId/10000 == 243, so this is impossible from a legitimate client. Not consuming.", s.CharacterId(), itemId)
			enableActions()
			return
		}

		templateId, err := itemInSlotFunc(l, ctx, s.CharacterId(), p.Source())
		if err != nil || templateId != uint32(itemId) {
			l.Warnf("Character [%d] attempted to use scripted item [%d] in slot [%d], but item not found or mismatched. Not consuming.", s.CharacterId(), itemId, p.Source())
			enableActions()
			return
		}

		cd, err := scriptedItemDataFunc(l, ctx, uint32(itemId))
		if err != nil {
			l.WithError(err).Errorf("Character [%d] scripted item [%d]: unable to read consumable data. Not consuming.", s.CharacterId(), itemId)
			enableActions()
			return
		}
		if cd.Npc() == 0 {
			l.Warnf("Character [%d] scripted item [%d] resolves to npc 0; no avatar to render the dialogue with. Every 0243 item authors npc under its spec node — if atlas-data has not been re-ingested since that parser fix, this is expected and re-ingest is the fix. Not consuming.", s.CharacterId(), itemId)
			enableActions()
			return
		}

		f := s.Field()
		now := time.Now()
		transactionId := uuid.New()

		// Conversation FIRST, destroy SECOND. The two dominant failure modes —
		// no conversation authored, and character already in a conversation —
		// both fail step 1, so step 2 never runs and the item is intact.
		sg := saga.Saga{
			TransactionId: transactionId,
			SagaType:      saga.ScriptedItemUse,
			InitiatedBy:   "SCRIPTED_ITEM",
			Steps: []saga.Step{
				{
					StepId: "start_item_conversation",
					Status: saga.Pending,
					Action: saga.StartItemConversation,
					Payload: saga.StartItemConversationPayload{
						CharacterId:   s.CharacterId(),
						AccountId:     s.AccountId(),
						ItemId:        uint32(itemId),
						NpcTemplateId: cd.Npc(),
						Slot:          p.Source(),
						WorldId:       f.WorldId(),
						ChannelId:     f.ChannelId(),
						MapId:         f.MapId(),
						Instance:      f.Instance(),
					},
					CreatedAt: now,
					UpdatedAt: now,
				},
				{
					StepId: "consume_scripted_item",
					Status: saga.Pending,
					Action: saga.DestroyAssetFromSlot,
					Payload: saga.DestroyAssetFromSlotPayload{
						CharacterId:   s.CharacterId(),
						InventoryType: byte(inventory.TypeValueUse),
						Slot:          p.Source(),
						Quantity:      1,
						ShowEffect:    false,
						TemplateId:    uint32(itemId),
					},
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		}

		if err := scriptedItemSagaCreateFunc(l, ctx, sg); err != nil {
			l.WithError(err).Errorf("Character [%d] scripted item [%d]: unable to create saga. Not consuming.", s.CharacterId(), itemId)
			enableActions()
			return
		}

		l.WithFields(logrus.Fields{
			"character_id":    s.CharacterId(),
			"item_id":         uint32(itemId),
			"slot":            p.Source(),
			"npc_template_id": cd.Npc(),
			"script_name":     cd.Script(),
			"transaction_id":  transactionId.String(),
		}).Info("Scripted item conversation requested.")
	}
}
