package characterstatus

import (
	"atlas-buffs/berserk"
	"atlas-buffs/character"
	consumer2 "atlas-buffs/kafka/consumer"
	characterstatus2 "atlas-buffs/kafka/message/characterstatus"
	"context"

	"github.com/segmentio/kafka-go"
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
			rf(consumer2.NewConfig(l)("character_status_event")(characterstatus2.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(characterstatus2.EnvEventTopicCharacterStatus)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogin))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventStatChanged))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMapChanged))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventChannelChanged))); err != nil {
			return err
		}
		return nil
	}
}

func handleStatusEventLogin(l logrus.FieldLogger, ctx context.Context, e characterstatus2.StatusEvent[characterstatus2.StatusEventLoginBody]) {
	if e.Type != characterstatus2.StatusEventTypeLogin {
		return
	}
	if err := berserk.NewProcessor(l, ctx).TrackOnLogin(e.WorldId, e.Body.ChannelId, e.CharacterId); err != nil {
		l.WithError(err).Errorf("Unable to evaluate berserk tracking for character [%d] at login.", e.CharacterId)
	}
}

func handleStatusEventLogout(l logrus.FieldLogger, ctx context.Context, e characterstatus2.StatusEvent[characterstatus2.StatusEventLogoutBody]) {
	if e.Type != characterstatus2.StatusEventTypeLogout {
		return
	}
	if err := berserk.NewProcessor(l, ctx).Untrack(e.CharacterId); err != nil {
		l.WithError(err).Errorf("Unable to untrack berserk for character [%d] at logout.", e.CharacterId)
	}
}

func handleStatusEventStatChanged(l logrus.FieldLogger, ctx context.Context, e characterstatus2.StatusEvent[characterstatus2.StatusEventStatChangedBody]) {
	if e.Type != characterstatus2.StatusEventTypeStatChanged {
		return
	}
	if err := berserk.NewProcessor(l, ctx).HandleStatChanged(e.WorldId, e.Body.ChannelId, e.CharacterId, e.Body.Updates); err != nil {
		l.WithError(err).Errorf("Unable to process stat change for berserk tracking of character [%d].", e.CharacterId)
	}
}

// handleStatusEventMapChanged handles two independent concerns on a map
// transition, each of which logs-and-continues on its own error so a
// failure in one never skips the other:
//
//  1. berserk transfer tracking (pre-existing).
//  2. Canceling any active HOMING_BEACON lock (Cosmic PlayerMapTransitionHandler
//     parity, design.md §2.2). The next map change or a logout/death
//     cancel-all is the safety net (design §5.7) if this cancel fails.
func handleStatusEventMapChanged(l logrus.FieldLogger, ctx context.Context, e characterstatus2.StatusEvent[characterstatus2.StatusEventMapChangedBody]) {
	if e.Type != characterstatus2.StatusEventTypeMapChanged {
		return
	}
	if err := berserk.NewProcessor(l, ctx).HandleTransfer(e.WorldId, e.Body.ChannelId, e.CharacterId); err != nil {
		l.WithError(err).Errorf("Unable to process map change for berserk tracking of character [%d].", e.CharacterId)
	}
	if err := character.NewProcessor(l, ctx).CancelByStatTypes(e.WorldId, e.CharacterId, []string{string(charconst.TemporaryStatTypeHomingBeacon)}); err != nil {
		l.WithError(err).Errorf("Unable to cancel HOMING_BEACON for character [%d] on map change.", e.CharacterId)
	}
}

func handleStatusEventChannelChanged(l logrus.FieldLogger, ctx context.Context, e characterstatus2.StatusEvent[characterstatus2.StatusEventChannelChangedBody]) {
	if e.Type != characterstatus2.StatusEventTypeChannelChanged {
		return
	}
	if err := berserk.NewProcessor(l, ctx).HandleTransfer(e.WorldId, e.Body.ChannelId, e.CharacterId); err != nil {
		l.WithError(err).Errorf("Unable to process channel change for berserk tracking of character [%d].", e.CharacterId)
	}
}
