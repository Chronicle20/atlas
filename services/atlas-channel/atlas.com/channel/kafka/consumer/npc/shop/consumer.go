package shop

import (
	"atlas-channel/character/skill"
	consumer2 "atlas-channel/kafka/consumer"
	shops2 "atlas-channel/kafka/message/npc/shop"
	"atlas-channel/npc/shops"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("npc_shop_status_event")(shops2.EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(shops2.EnvStatusEventTopic)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleEnteredStatusEvent(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleErrorStatusEvent(sc, wp))))
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
		bp := writer.NPCShopBody(l, tenant.MustFromContext(ctx))(e.Body.NpcTemplateId, nsm.Commodities(), sms)
		_ = session.Announce(l)(ctx)(wp)(writer.NPCShop)(bp)(s)
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

		var bp writer.BodyProducer
		if e.Body.Error == writer.NPCShopOperationOverLevelRequirement {
			bp = writer.NPCShopOperationOverLevelRequirementBody(l, t)(e.Body.LevelLimit)
		} else if e.Body.Error == writer.NPCShopOperationUnderLevelRequirement {
			bp = writer.NPCShopOperationUnderLevelRequirementBody(l, t)(e.Body.LevelLimit)
		} else if e.Body.Error == writer.NPCShopOperationGenericError {
			bp = writer.NPCShopOperationGenericErrorBody(l, t)
		} else if e.Body.Error == writer.NPCShopOperationGenericErrorWithReason {
			bp = writer.NPCShopOperationGenericErrorWithReasonBody(l, t)(e.Body.Reason)
		} else {
			bp = writer.NPCShopOperationBody(l, tenant.MustFromContext(ctx))(e.Body.Error)
		}
		_ = session.Announce(l)(ctx)(wp)(writer.NPCShopOperation)(bp)(s)
	}
}
