package food

import (
	"context"

	"atlas-consumables/consumable"
	consumer2 "atlas-consumables/kafka/consumer"
	foodmsg "atlas-consumables/kafka/message/food"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("taming_mob_food_command")(foodmsg.EnvCommandTopic)(consumerGroupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		t, _ := topic.EnvProvider(l)(foodmsg.EnvCommandTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleRequestFeed))); err != nil {
			return err
		}
		return nil
	}
}

func handleRequestFeed(l logrus.FieldLogger, ctx context.Context, c foodmsg.Command[foodmsg.RequestFeedBody]) {
	if c.Type != foodmsg.CommandRequestFeed {
		return
	}
	itemId := item.Id(c.Body.ItemId)
	if item.GetClassification(itemId) != item.ClassificationRevitalizer {
		l.Warnf("Character [%d] requested taming-mob feed with non-revitalizer item [%d] (classification [%d]). Rejecting.", c.CharacterId, c.Body.ItemId, item.GetClassification(itemId))
		return
	}
	err := consumable.NewProcessor(l, ctx).RequestFeed(c.WorldId, uint32(c.CharacterId), c.Body.Slot, itemId)
	if err != nil {
		l.WithError(err).Errorf("Character [%d] unable to feed taming-mob with item [%d] in slot [%d] as expected.", c.CharacterId, c.Body.ItemId, c.Body.Slot)
	}
}
