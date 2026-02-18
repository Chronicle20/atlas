package character

import (
	consumer2 "atlas-party-quests/kafka/consumer"
	"atlas-party-quests/instance"
	character2 "atlas-party-quests/kafka/message/character"
	"context"

	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status_event")(character2.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger, db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(rf func(topic string, handler handler.Handler) (string, error)) {
		var t string
		t, _ = topic.EnvProvider(l)(character2.EnvEventTopicCharacterStatus)()
		_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout(db))))
	}
}

func handleStatusEventLogout(db *gorm.DB) message.Handler[character2.StatusEvent[character2.StatusEventLogoutBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e character2.StatusEvent[character2.StatusEventLogoutBody]) {
		if e.Type != character2.StatusEventTypeLogout {
			return
		}

		l.Debugf("Processing logout event for character [%d].", e.CharacterId)

		err := instance.NewProcessor(l, ctx, db).LeaveAndEmit(e.CharacterId, "disconnect")
		if err != nil {
			// Character may not be in a PQ — this is expected and not an error.
			l.Debugf("Character [%d] not in active PQ or unable to leave: %s.", e.CharacterId, err.Error())
			return
		}

		l.Infof("Character [%d] automatically left PQ due to logout.", e.CharacterId)
	}
}
