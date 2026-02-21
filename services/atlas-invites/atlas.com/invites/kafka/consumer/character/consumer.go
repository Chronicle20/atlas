package character

import (
	invite "atlas-invites/invite"
	consumer2 "atlas-invites/kafka/consumer"
	character2 "atlas-invites/kafka/message/character"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status_event")(character2.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(character2.EnvEventTopicCharacterStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDeleted))); err != nil {
			return err
		}
		return nil
	}
}

func handleStatusEventDeleted(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventDeletedBody]) {
	if e.Type != character2.StatusEventTypeDeleted {
		return
	}
	err := invite.NewProcessor(l, ctx).DeleteByCharacterIdAndEmit(e.CharacterId)
	if err != nil {
		l.WithError(err).Errorf("Unable to delete invites for character [%d].", e.CharacterId)
	}
}
