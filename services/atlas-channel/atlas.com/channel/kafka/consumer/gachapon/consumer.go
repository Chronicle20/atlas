package gachapon

import (
	consumer2 "atlas-channel/kafka/consumer"
	gachapon2 "atlas-channel/kafka/message/gachapon"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("gachapon_reward_won")(gachapon2.EnvEventTopicGachaponRewardWon)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(gachapon2.EnvEventTopicGachaponRewardWon)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleRewardWon(sc, wp))))
			}
		}
	}
}

func handleRewardWon(sc server.Model, wp writer.Producer) message.Handler[gachapon2.RewardWonEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, event gachapon2.RewardWonEvent) {
		t := tenant.MustFromContext(ctx)
		if !sc.IsWorld(t, world.Id(event.WorldId)) {
			return
		}

		l.WithFields(logrus.Fields{
			"character_name": event.CharacterName,
			"item_id":        event.ItemId,
			"tier":           event.Tier,
			"gachapon_name":  event.GachaponName,
		}).Infof("Broadcasting gachapon reward won.")

		itemOperator := func(w *response.Writer) error {
			w.WriteInt(event.ItemId)
			return nil
		}

		bodyProducer := writer.WorldMessageGachaponMegaphoneBody(l)("", event.CharacterName, sc.ChannelId(), event.GachaponName, itemOperator)

		sessions, err := session.NewProcessor(l, ctx).AllInTenantProvider()
		if err != nil {
			l.WithError(err).Error("Unable to get sessions for gachapon broadcast.")
			return
		}

		announceOp := session.Announce(l)(ctx)(wp)(writer.WorldMessage)(bodyProducer)
		for _, s := range sessions {
			err := announceOp(s)
			if err != nil {
				l.WithError(err).Warnf("Unable to send gachapon announcement to session.")
			}
		}
	}
}
