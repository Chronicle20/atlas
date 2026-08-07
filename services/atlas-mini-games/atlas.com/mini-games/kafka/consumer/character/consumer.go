package character

import (
	"atlas-mini-games/game"
	consumer2 "atlas-mini-games/kafka/consumer"
	characterKafka "atlas-mini-games/kafka/message/character"
	"context"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("status_event")(characterKafka.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			t, _ = topic.EnvProvider(l)(characterKafka.EnvEventTopicCharacterStatus)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventLogout(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMapChanged(db)))); err != nil {
				return err
			}
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventChannelChanged(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

// handleStatusEventLogout tears down whatever mini-game room the character
// occupies when they log out, same forfeit-then-leave path as an explicit
// LEAVE command.
func handleStatusEventLogout(db *gorm.DB) message.Handler[characterKafka.StatusEvent[characterKafka.StatusEventLogoutBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e characterKafka.StatusEvent[characterKafka.StatusEventLogoutBody]) {
		if e.Type != characterKafka.EventCharacterStatusTypeLogout {
			return
		}
		l.Debugf("Character [%d] has logged out. worldId [%d] channelId [%d] mapId [%d]. Tearing down mini-game membership.", e.CharacterId, e.WorldId, e.Body.ChannelId, e.Body.MapId)
		if err := game.NewProcessor(l, ctx, db).TeardownCharacter(e.CharacterId); err != nil {
			l.WithError(err).Errorf("Unable to tear down mini-game membership for character [%d] on logout.", e.CharacterId)
		}
	}
}

// handleStatusEventMapChanged tears down whatever mini-game room the
// character occupies when they leave the map it lives in (a mini-game room
// is field-scoped, so a map transition ends the character's membership).
func handleStatusEventMapChanged(db *gorm.DB) message.Handler[characterKafka.StatusEvent[characterKafka.StatusEventMapChangedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e characterKafka.StatusEvent[characterKafka.StatusEventMapChangedBody]) {
		if e.Type != characterKafka.EventCharacterStatusTypeMapChanged {
			return
		}
		l.Debugf("Character [%d] has changed maps. worldId [%d] channelId [%d] oldMapId [%d] newMapId [%d]. Tearing down mini-game membership.", e.CharacterId, e.WorldId, e.Body.ChannelId, e.Body.OldMapId, e.Body.TargetMapId)
		if err := game.NewProcessor(l, ctx, db).TeardownCharacter(e.CharacterId); err != nil {
			l.WithError(err).Errorf("Unable to tear down mini-game membership for character [%d] on map change.", e.CharacterId)
		}
	}
}

// handleStatusEventChannelChanged tears down whatever mini-game room the
// character occupies when they switch channels (chalkboards precedent,
// services/atlas-chalkboards/atlas.com/chalkboards/kafka/consumer/character/
// consumer.go:75-83). A channel change emits neither LOGOUT nor MAP_CHANGED,
// so without this the member index would keep the character bound to the
// stale old-channel room — Create returns ErrOwnerHasRoom and Visit returns
// UNABLE indefinitely, and leave/chat/ready commands from the new channel
// would resolve against the stale room.
func handleStatusEventChannelChanged(db *gorm.DB) message.Handler[characterKafka.StatusEvent[characterKafka.ChangeChannelEventLoginBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e characterKafka.StatusEvent[characterKafka.ChangeChannelEventLoginBody]) {
		if e.Type != characterKafka.EventCharacterStatusTypeChannelChanged {
			return
		}
		l.Debugf("Character [%d] has changed channels. worldId [%d] channelId [%d] oldChannelId [%d]. Tearing down mini-game membership.", e.CharacterId, e.WorldId, e.Body.ChannelId, e.Body.OldChannelId)
		if err := game.NewProcessor(l, ctx, db).TeardownCharacter(e.CharacterId); err != nil {
			l.WithError(err).Errorf("Unable to tear down mini-game membership for character [%d] on channel change.", e.CharacterId)
		}
	}
}
