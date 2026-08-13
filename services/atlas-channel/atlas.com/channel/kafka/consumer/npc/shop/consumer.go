package shop

import (
	"atlas-channel/character/skill"
	consumer2 "atlas-channel/kafka/consumer"
	shops2 "atlas-channel/kafka/message/npc/shop"
	"atlas-channel/listener"
	"atlas-channel/npc/shops"
	"atlas-channel/remotemerchant"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	npcpkt "github.com/Chronicle20/atlas/libs/atlas-packet/npc/clientbound"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("npc_shop_status_event")(shops2.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(ctx context.Context) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(ctx context.Context) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
					var t string
					var handles []listener.HandlerHandle
					t, _ = topic.EnvProvider(l)(shops2.EnvStatusEventTopic)()
					id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleEnteredStatusEvent(sc, wp))))
					if err != nil {
						return nil, err
					}
					handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
					id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleErrorStatusEvent(sc, wp))))
					if err != nil {
						return nil, err
					}
					handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
					id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleEnterErrorStatusEvent(sc, wp))))
					if err != nil {
						return nil, err
					}
					handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})

					startRemoteMerchantSweep(l, ctx, sc, wp)

					return handles, nil
				}
			}
		}
	}
}

// unlockPendingRemoteMerchant sends EnableActions if and only if this character
// reached the shop by using a classification-545 cash item. The client sets
// m_bExclRequestSent when it sends CASH_ITEM_USE and CShopDlg::SetShopDlg never
// clears it (task-221 design §1.2 OQ-2), so the server must unlock — but only
// for that path. Unlocking the ordinary NPC-talk path would change the bytes on
// versions whose OPEN_NPC_SHOP cells are already verified.
func unlockPendingRemoteMerchant(l logrus.FieldLogger, t tenant.Model, characterId uint32, unlock func()) {
	e, ok := remotemerchant.GetRegistry().Take(t, characterId)
	if !ok {
		return
	}
	l.WithFields(logrus.Fields{
		"character_id": characterId,
		"item_id":      uint32(e.ItemId),
		"slot":         int16(e.Slot),
	}).Debug("Unlocking client after a remote-merchant shop open.")
	unlock()
}

func handleEnteredStatusEvent(sc server.Model, wp writer.Producer) message.Handler[shops2.StatusEvent[shops2.StatusEventEnteredBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e shops2.StatusEvent[shops2.StatusEventEnteredBody]) {
		if e.Type != shops2.StatusEventTypeEntered {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		s, err := session.NewProcessor(l, ctx).GetByCharacterId(sc.Channel())(e.CharacterId)
		if err != nil {
			return
		}

		// EnableActions is the client's duplicate-request gate, so it goes
		// AFTER the shop packet on the success path — but it must still fire if
		// the announce below bails out, or the player is locked for nothing.
		defer unlockPendingRemoteMerchant(l, t, e.CharacterId, func() {
			_ = session.EnableActions(l)(ctx)(wp)(s)
		})

		sms, err := skill.NewProcessor(l, ctx).GetByCharacterId(s.CharacterId())
		if err != nil {
			l.WithError(err).Errorf("Unable to get skills for character [%d].", s.CharacterId())
			return
		}

		nsm, err := shops.NewProcessor(l, ctx).GetShop(e.Body.NpcTemplateId)
		if err != nil {
			l.WithError(err).Errorf("Unable to get shop for NPC [%d].", e.Body.NpcTemplateId)
			return
		}
		set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion())
		bp := writer.NPCShopBody(e.Body.NpcTemplateId, nsm.Commodities(), sms, set.Skill)
		_ = session.Announce(l)(ctx)(wp)(npcpkt.NPCShopWriter)(bp)(s)
	}
}

func handleErrorStatusEvent(sc server.Model, wp writer.Producer) message.Handler[shops2.StatusEvent[shops2.StatusEventErrorBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e shops2.StatusEvent[shops2.StatusEventErrorBody]) {
		if e.Type != shops2.StatusEventTypeError {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		s, err := session.NewProcessor(l, ctx).GetByCharacterId(sc.Channel())(e.CharacterId)
		if err != nil {
			return
		}

		var bp packet.Encode
		switch e.Body.Error {
		case npcpkt.NPCShopOperationOk:
			bp = npcpkt.NPCShopOperationOkBody()
		case npcpkt.NPCShopOperationOutOfStock:
			bp = npcpkt.NPCShopOperationOutOfStockBody()
		case npcpkt.NPCShopOperationNotEnoughMoney:
			bp = npcpkt.NPCShopOperationNotEnoughMoneyBody()
		case npcpkt.NPCShopOperationInventoryFull:
			bp = npcpkt.NPCShopOperationInventoryFullBody()
		case npcpkt.NPCShopOperationOutOfStock2:
			bp = npcpkt.NPCShopOperationOutOfStock2Body()
		case npcpkt.NPCShopOperationOutOfStock3:
			bp = npcpkt.NPCShopOperationOutOfStock3Body()
		case npcpkt.NPCShopOperationNotEnoughMoney2:
			bp = npcpkt.NPCShopOperationNotEnoughMoney2Body()
		case npcpkt.NPCShopOperationNeedMoreItems:
			bp = npcpkt.NPCShopOperationNeedMoreItemsBody()
		case npcpkt.NPCShopOperationTradeLimit:
			bp = npcpkt.NPCShopOperationTradeLimitBody()
		case npcpkt.NPCShopOperationOverLevelRequirement:
			bp = npcpkt.NPCShopOperationOverLevelRequirementBody(e.Body.LevelLimit)
		case npcpkt.NPCShopOperationUnderLevelRequirement:
			bp = npcpkt.NPCShopOperationUnderLevelRequirementBody(e.Body.LevelLimit)
		case npcpkt.NPCShopOperationGenericError:
			bp = npcpkt.NPCShopOperationGenericErrorBody()
		case npcpkt.NPCShopOperationGenericErrorWithReason:
			bp = npcpkt.NPCShopOperationGenericErrorWithReasonBody(e.Body.Reason)
		default:
			l.Warnf("Unhandled NPC shop operation error code [%s].", e.Body.Error)
			return
		}
		_ = session.Announce(l)(ctx)(wp)(npcpkt.NPCShopOperationWriter)(bp)(s)
	}
}

// handleEnterErrorStatusEvent unlocks the client after a failed remote-merchant
// shop open. It deliberately writes NO packet: NPCShopOperation with no
// outstanding buy/sell/recharge request throws CDisconnectException in
// CShopDlg::OnPacket (@0x756da7), which is exactly why ENTER_ERROR is a
// separate status type from ERROR (task-221 design delta D5).
func handleEnterErrorStatusEvent(sc server.Model, wp writer.Producer) message.Handler[shops2.StatusEvent[shops2.StatusEventEnterErrorBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e shops2.StatusEvent[shops2.StatusEventEnterErrorBody]) {
		if e.Type != shops2.StatusEventTypeEnterError {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		s, err := session.NewProcessor(l, ctx).GetByCharacterId(sc.Channel())(e.CharacterId)
		if err != nil {
			return
		}

		l.WithFields(logrus.Fields{
			"character_id":    e.CharacterId,
			"npc_template_id": e.Body.NpcTemplateId,
			"reason":          e.Body.Reason,
		}).Warn("NPC shop enter failed.")

		unlockPendingRemoteMerchant(l, t, e.CharacterId, func() {
			_ = session.EnableActions(l)(ctx)(wp)(s)
		})
	}
}

// startRemoteMerchantSweep evicts pending unlocks whose status event never
// arrived and unlocks those clients, so a dropped event cannot leave a
// character permanently locked (task-221 design §2.3).
func startRemoteMerchantSweep(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer) {
	routine.Go(l, ctx, func(c context.Context) {
		ticker := time.NewTicker(remotemerchant.TTL)
		defer ticker.Stop()
		for {
			select {
			case <-c.Done():
				return
			case now := <-ticker.C:
				for _, ex := range remotemerchant.GetRegistry().Sweep(now) {
					if !ex.Tenant.Is(sc.Tenant()) {
						continue
					}
					s, err := session.NewProcessor(l, c).GetByCharacterId(sc.Channel())(ex.CharacterId)
					if err != nil {
						continue
					}
					l.WithFields(logrus.Fields{
						"character_id": ex.CharacterId,
						"item_id":      uint32(ex.Entry.ItemId),
					}).Warn("Remote-merchant shop open timed out with no status event; unlocking client.")
					_ = session.EnableActions(l)(c)(wp)(s)
				}
			}
		}
	})
}
