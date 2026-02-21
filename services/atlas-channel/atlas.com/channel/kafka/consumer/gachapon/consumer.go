package gachapon

import (
	"atlas-channel/character"
	"atlas-channel/compartment"
	consumer2 "atlas-channel/kafka/consumer"
	gachapon2 "atlas-channel/kafka/message/gachapon"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
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

		inventoryType, ok := inventory.TypeFromItemId(item.Id(event.ItemId))
		if !ok {
			l.Errorf("Unable to identify inventory type for item [%d].", event.ItemId)
			return
		}

		comp, err := compartment.NewProcessor(l, ctx).GetByType(event.CharacterId, inventoryType)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve compartment for character [%d] inventory type [%d].", event.CharacterId, inventoryType)
			return
		}

		a, found := comp.FindById(event.AssetId)
		if !found {
			l.Errorf("Unable to find asset [%d] for character [%d].", event.AssetId, event.CharacterId)
			return
		}

		itemOperator := func(w *response.Writer) error {
			return writer.WriteAssetInfo(t)(true)(w)(*a)
		}

		l.WithFields(logrus.Fields{
			"character_name": c.Name(),
			"item_id":        event.ItemId,
			"tier":           event.Tier,
			"gachapon_name":  event.GachaponName,
		}).Infof("Broadcasting gachapon reward won.")

		bodyProducer := writer.WorldMessageGachaponMegaphoneBody(l)("", c.Name(), sc.ChannelId(), event.GachaponName, itemOperator)

		sessions, err := session.NewProcessor(l, ctx).AllInChannelProvider(sc.WorldId(), sc.ChannelId())
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
