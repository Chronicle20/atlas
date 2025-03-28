package character

import (
	consumer2 "atlas-skills/kafka/consumer"
	character2 "atlas-skills/kafka/message/character"
	"atlas-skills/skill"
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

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(character2.EnvEventTopicCharacterStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout)))
	}
}

func handleStatusEventLogout(l logrus.FieldLogger, ctx context.Context, event character2.StatusEvent[character2.StatusEventLogoutBody]) {
	if event.Type != character2.EventCharacterStatusTypeLogout {
		return
	}
	err := skill.ClearAll(ctx)(event.CharacterId)
	if err != nil {
		l.WithError(err).Errorf("Unable to process logout for character [%d].", event.CharacterId)
	}
}
