package character

import (
	"atlas-portals/blocked"
	consumer2 "atlas-portals/kafka/consumer"
	characterKafka "atlas-portals/kafka/message/character"
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
			rf(consumer2.NewConfig(l)("character_status_event")(characterKafka.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(characterKafka.EnvEventTopicCharacterStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout)))
	}
}

func handleStatusEventLogout(l logrus.FieldLogger, ctx context.Context, event characterKafka.StatusEvent[characterKafka.StatusEventLogoutBody]) {
	if event.Type != characterKafka.EventCharacterStatusTypeLogout {
		return
	}

	l.Debugf("Character [%d] has logged out. Clearing blocked portals.", event.CharacterId)
	blocked.GetRegistry().ClearForCharacter(ctx, event.CharacterId)
}
