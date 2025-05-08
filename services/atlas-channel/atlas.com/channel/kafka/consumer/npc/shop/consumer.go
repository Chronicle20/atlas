package shop

import (
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
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleShopStatusEvent(l, sc, wp))))
			}
		}
	}
}

func handleShopStatusEvent(l logrus.FieldLogger, sc server.Model, wp writer.Producer) message.Handler[shops2.StatusEvent[shops2.StatusEventEnteredBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e shops2.StatusEvent[shops2.StatusEventEnteredBody]) {
		if e.Type != shops2.StatusEventTypeEntered {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !t.Is(sc.Tenant()) {
			return
		}

		s, err := session.NewProcessor(l, ctx).GetByCharacterId(sc.WorldId(), sc.ChannelId())(e.CharacterId)
		if err != nil {
			return
		}
		p := shops.NewProcessor(l, ctx)
		nsm, err := p.GetShop(e.Body.NpcTemplateId)
		if err != nil {
			l.WithError(err).Errorf("Unable to get shop for NPC [%d].", e.Body.NpcTemplateId)
			return
		}
		bp := writer.NPCShopBody(l, tenant.MustFromContext(ctx))(e.Body.NpcTemplateId, nsm.Commodities())
		_ = session.Announce(l)(ctx)(wp)(writer.NPCShop)(bp)(s)
	}
}
