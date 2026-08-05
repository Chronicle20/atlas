package charstatus

import (
	"atlas-buffs/character"
	consumer2 "atlas-buffs/kafka/consumer"
	"context"

	"github.com/sirupsen/logrus"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_status_event")(EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		t, _ := topic.EnvProvider(l)(EnvEventTopicCharacterStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleMapChanged))); err != nil {
			return err
		}
		return nil
	}
}

// handleMapChanged cancels any active HOMING_BEACON lock when the character
// completes a map transition (Cosmic PlayerMapTransitionHandler parity —
// design.md §2.2). Errors log-and-continue; the next map change or a
// logout/death cancel-all is the safety net (design §5.7).
func handleMapChanged(l logrus.FieldLogger, ctx context.Context, e StatusEvent[MapChangedBody]) {
	if e.Type != StatusEventTypeMapChanged {
		return
	}
	l.Debugf("Character [%d] changed maps [%d] -> [%d]; canceling HOMING_BEACON if present.", e.CharacterId, e.Body.OldMapId, e.Body.TargetMapId)
	if err := character.NewProcessor(l, ctx).CancelByStatTypes(e.WorldId, e.CharacterId, []string{string(charconst.TemporaryStatTypeHomingBeacon)}); err != nil {
		l.WithError(err).Errorf("Unable to cancel HOMING_BEACON for character [%d] on map change.", e.CharacterId)
	}
}
