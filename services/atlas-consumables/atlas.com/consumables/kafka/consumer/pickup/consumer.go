package pickup

import (
	"context"

	consumer2 "atlas-consumables/kafka/consumer"
	mbmsg "atlas-consumables/kafka/message/monsterbook"
	pickupmsg "atlas-consumables/kafka/message/pickup"
	"atlas-consumables/kafka/producer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	kmessage "github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	kafkaProducer "github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("item_consumed_on_pickup")(pickupmsg.EnvCommandTopic)(consumerGroupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		t, _ := topic.EnvProvider(l)(pickupmsg.EnvCommandTopic)()
		if _, err := rf(t, kmessage.AdaptHandler(kmessage.PersistentConfig(handlePickup))); err != nil {
			return err
		}
		return nil
	}
}

func handlePickup(l logrus.FieldLogger, ctx context.Context, cmd pickupmsg.Command) {
	if cmd.Type != pickupmsg.CommandType {
		return
	}
	if item.GetClassification(item.Id(cmd.ItemId)) != item.ClassificationConsumableMonsterCard {
		l.Warnf("ITEM.CONSUMED_ON_PICKUP for non-card item %d - no handler yet, skipping.", cmd.ItemId)
		return
	}

	t, err := topic.EnvProvider(l)(mbmsg.EnvCommandTopic)()
	if err != nil {
		l.WithError(err).Errorf("Unable to resolve monster book command topic.")
		return
	}

	if err := producer.ProviderImpl(l)(ctx)(t)(cardPickedUpProvider(cmd)); err != nil {
		l.WithError(err).Errorf("Failed to emit MONSTER_BOOK.CARD_PICKED_UP for character %d card %d.", cmd.CharacterId, cmd.ItemId)
	}
}

func cardPickedUpProvider(cmd pickupmsg.Command) model.Provider[[]kafka.Message] {
	key := kafkaProducer.CreateKey(int(cmd.CharacterId))
	value := &mbmsg.Command[mbmsg.CardPickedUpBody]{
		TenantId:    cmd.TenantId,
		CharacterId: cmd.CharacterId,
		EventId:     cmd.TransactionId, // transactionId is the eventId per design Section 4.1
		Type:        mbmsg.CommandTypeCardPickedUp,
		Body: mbmsg.CardPickedUpBody{
			CardId: cmd.ItemId,
			Source: "drop_pickup",
		},
	}
	return kafkaProducer.SingleMessageProvider(key, value)
}
