package gachapon

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	gachapon2 "atlas-channel/kafka/message/gachapon"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	chatpkt "github.com/Chronicle20/atlas/libs/atlas-packet/chat/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
			return func(rf func(topic string, handler handler.Handler) (string, error)) error {
				var t string
				t, _ = topic.EnvProvider(l)(gachapon2.EnvEventTopicGachaponRewardWon)()
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRewardWon(sc, wp)))); err != nil {
					return err
				}
				return nil
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

		c, err := character.NewProcessor(l, ctx).GetById()(event.CharacterId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve character [%d] for gachapon broadcast.", event.CharacterId)
			return
		}

		l.WithFields(logrus.Fields{
			"character_name": c.Name(),
			"item_id":        event.ItemId,
			"tier":           event.Tier,
			"gachapon_name":  event.GachaponName,
		}).Infof("Broadcasting gachapon reward won.")

		sessions, err := session.NewProcessor(l, ctx).AllInChannelProvider(sc.WorldId(), sc.ChannelId())
		if err != nil {
			l.WithError(err).Error("Unable to get sessions for gachapon broadcast.")
			return
		}

		announceOp := session.Announce(l)(ctx)(wp)(chatpkt.WorldMessageWriter)(func(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
			return func(options map[string]interface{}) []byte {
				return writer.WorldMessageGachaponMegaphoneBody("", c.Name(), sc.ChannelId(), event.GachaponName, event.ItemId)(l, ctx)(options)
			}
		})
		for _, s := range sessions {
			err = announceOp(s)
			if err != nil {
				l.WithError(err).Warnf("Unable to send gachapon announcement to session.")
			}
		}
	}
}
