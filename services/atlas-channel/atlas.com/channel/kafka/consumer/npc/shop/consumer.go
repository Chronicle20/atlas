package shop

import (
	"atlas-channel/character/skill"
	consumer2 "atlas-channel/kafka/consumer"
	shops2 "atlas-channel/kafka/message/npc/shop"
	"atlas-channel/listener"
	"atlas-channel/npc/shops"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	npcpkt "github.com/Chronicle20/atlas/libs/atlas-packet/npc/clientbound"
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

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
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
				return handles, nil
			}
		}
	}
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
		bp := writer.NPCShopBody(e.Body.NpcTemplateId, nsm.Commodities(), sms)
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
