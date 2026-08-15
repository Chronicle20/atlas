package handler

import (
	"atlas-channel/npc/shops"
	"atlas-channel/remotemerchant"
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	invsb "github.com/Chronicle20/atlas/libs/atlas-packet/inventory/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// npcItemUseSagaCreateFunc is a test seam for saga creation (precedent:
// scriptedItemSagaCreateFunc, scripted_item.go).
var npcItemUseSagaCreateFunc = func(l logrus.FieldLogger, ctx context.Context, s saga.Saga) error {
	return saga.NewProcessor(l, ctx).Create(s)
}

// npcShopProbeFunc is a test seam over the shop-vs-conversation probe this
// handler shares with NPCStartConversationHandleFunc
// (npc_start_conversation.go:31-42): err == nil means npcTemplateId has a
// shop, err != nil means it does not. Kept as a seam (rather than an inline
// call to shops.NewProcessor(...).GetShop) purely for testability — the
// predicate itself is unchanged.
var npcShopProbeFunc = func(l logrus.FieldLogger, ctx context.Context, npcTemplateId uint32) error {
	_, err := shops.NewProcessor(l, ctx).GetShop(npcTemplateId)
	return err
}

// NpcItemUseHandleFunc handles CWvsContext::SendSelectNpcItemUseRequest, which
// covers two item families that share one opcode:
//
//	239xxxx — remote-NPC summons. The item names an NPC in its info/npc node;
//	          open that NPC's shop if it has one, otherwise its conversation.
//	545xxxx — remote merchant. Always a shop.
//
// Dispatch is classification-first, never slot-type-first, for the reason
// character_cash_item_use.go already documents: type bytes collide.
//
// Interaction with the CASH_ITEM_USE route (task-221): on v72-v95 both opcodes
// exist and CDraggableItem::OnDoubleClicked decides which one a 545 item goes
// out as. The server accepts BOTH; neither handler may assume it is the only
// path. On v61 this is the ONLY route for 545 — remoteMerchantEnabled() is
// correctly false there because 545 sits in that version's CASH_ITEM_USE
// dispatcher default arm — so this handler must not consult that predicate.
func NpcItemUseHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := invsb.NpcItemUse{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		enableActions := func() { _ = session.EnableActions(l)(ctx)(wp)(s) }

		itemId := item.Id(p.ItemId())

		switch item.GetClassification(itemId) {
		case item.ClassificationConsumableRemoteNpc:
			handleRemoteNpcItemUse(l, ctx, s, p, itemId, enableActions)
		case item.ClassificationRemoteMerchant:
			handleRemoteMerchantItemUse(l, ctx, s, p, itemId, enableActions)
		default:
			l.Warnf("Character [%d] attempted npc item use with item [%d] of an unhandled classification. The client gates this op on itemId/10000 in {239, 545}, so this is impossible from a legitimate client. Not consuming.", s.CharacterId(), itemId)
			enableActions()
		}
	}
}

// handleRemoteNpcItemUse implements classification 239 (remote-NPC summon):
// the item names an NPC and opens that NPC's shop if it has one, otherwise
// its conversation — the same probe NPCStartConversationHandleFunc performs
// for a live map object (npc_start_conversation.go:31-42), reused here rather
// than reinvented.
func handleRemoteNpcItemUse(l logrus.FieldLogger, ctx context.Context, s session.Model, p invsb.NpcItemUse, itemId item.Id, enableActions func()) {
	templateId, err := itemInSlotFunc(l, ctx, s.CharacterId(), p.Source())
	if err != nil || templateId != uint32(itemId) {
		l.Warnf("Character [%d] attempted to use npc item [%d] in slot [%d], but item not found or mismatched. Not consuming.", s.CharacterId(), itemId, p.Source())
		enableActions()
		return
	}

	cd, err := scriptedItemDataFunc(l, ctx, uint32(itemId))
	if err != nil {
		l.WithError(err).Errorf("Character [%d] npc item [%d]: unable to read consumable data. Not consuming.", s.CharacterId(), itemId)
		enableActions()
		return
	}
	if cd.Npc() == 0 {
		l.Warnf("Character [%d] npc item [%d] resolves to npc 0; no avatar to open a shop or conversation with. Not consuming.", s.CharacterId(), itemId)
		enableActions()
		return
	}

	f := s.Field()
	now := time.Now()
	transactionId := uuid.New()

	shopErr := npcShopProbeFunc(l, ctx, cd.Npc())
	route := "conversation"
	var firstStep saga.Step
	if shopErr == nil {
		route = "shop"
		firstStep = saga.Step{
			StepId: "open_npc_shop",
			Status: saga.Pending,
			Action: saga.OpenNpcShop,
			Payload: saga.OpenNpcShopPayload{
				CharacterId:   s.CharacterId(),
				WorldId:       f.WorldId(),
				ChannelId:     f.ChannelId(),
				NpcTemplateId: cd.Npc(),
			},
			CreatedAt: now,
			UpdatedAt: now,
		}
	} else {
		firstStep = saga.Step{
			StepId: "start_npc_conversation",
			Status: saga.Pending,
			Action: saga.StartNpcConversation,
			Payload: saga.StartNpcConversationPayload{
				CharacterId:   s.CharacterId(),
				AccountId:     s.AccountId(),
				NpcTemplateId: cd.Npc(),
				WorldId:       f.WorldId(),
				ChannelId:     f.ChannelId(),
				MapId:         f.MapId(),
				Instance:      f.Instance(),
			},
			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	sg := saga.Saga{
		TransactionId: transactionId,
		SagaType:      saga.RemoteNpcUse,
		InitiatedBy:   "NPC_ITEM_USE",
		Steps: []saga.Step{
			firstStep,
			{
				StepId: "consume_remote_npc_item",
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

	if err := npcItemUseSagaCreateFunc(l, ctx, sg); err != nil {
		l.WithError(err).Errorf("Character [%d] npc item [%d]: unable to create saga. Not consuming.", s.CharacterId(), itemId)
		enableActions()
		return
	}

	l.WithFields(logrus.Fields{
		"character_id":    s.CharacterId(),
		"item_id":         uint32(itemId),
		"slot":            p.Source(),
		"npc_template_id": cd.Npc(),
		"route":           route,
		"transaction_id":  transactionId.String(),
	}).Info("Remote NPC item use requested.")
}

// handleRemoteMerchantItemUse implements classification 545 (remote
// merchant): always a shop. Ownership is validated in the CASH inventory
// (character_cash_item_use.go:45-58's lookup, mirrored via
// cashItemInSlotFunc), not the USE inventory this route's 239 sibling reads,
// because this item lives in the cash tab.
func handleRemoteMerchantItemUse(l logrus.FieldLogger, ctx context.Context, s session.Model, p invsb.NpcItemUse, itemId item.Id, enableActions func()) {
	templateId, err := cashItemInSlotFunc(l, ctx, s.CharacterId(), p.Source())
	if err != nil || templateId != uint32(itemId) {
		l.Warnf("Character [%d] attempted to use remote merchant item [%d] in slot [%d], but item not found or mismatched. Not consuming.", s.CharacterId(), itemId, p.Source())
		enableActions()
		return
	}

	ci, err := cashItemDataFunc(l, ctx, uint32(itemId))
	if err != nil {
		l.WithError(err).Errorf("Character [%d] remote merchant item [%d]: unable to read cash item data. Not consuming.", s.CharacterId(), itemId)
		enableActions()
		return
	}
	if ci.Npc == 0 {
		l.Warnf("Character [%d] remote merchant item [%d] resolves to npc 0; no shop to open. Not consuming.", s.CharacterId(), itemId)
		enableActions()
		return
	}

	t := tenant.MustFromContext(ctx)
	f := s.Field()
	now := time.Now()
	transactionId := uuid.New()

	// Registry first: a very fast ENTERED must not arrive before the entry
	// that tells the shop consumer to unlock this client
	// (character_cash_item_use_remote_merchant.go:113-117).
	remotemerchant.GetRegistry().Put(t, s.CharacterId(), remotemerchant.Entry{
		ItemId: itemId,
		Slot:   slot.Position(p.Source()),
		At:     now,
	})

	sg := saga.Saga{
		TransactionId: transactionId,
		SagaType:      saga.RemoteMerchant,
		InitiatedBy:   "NPC_ITEM_USE",
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
					InventoryType: byte(inventory.TypeValueCash),
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

	if err := npcItemUseSagaCreateFunc(l, ctx, sg); err != nil {
		l.WithError(err).Errorf("Character [%d] remote merchant item [%d]: unable to create saga. Not consuming.", s.CharacterId(), itemId)
		remotemerchant.GetRegistry().ClearCharacter(t, s.CharacterId())
		enableActions()
		return
	}

	l.WithFields(logrus.Fields{
		"character_id":    s.CharacterId(),
		"item_id":         uint32(itemId),
		"slot":            p.Source(),
		"npc_template_id": ci.Npc,
		"route":           "shop",
		"transaction_id":  transactionId.String(),
	}).Info("Remote merchant shop open requested.")
}
