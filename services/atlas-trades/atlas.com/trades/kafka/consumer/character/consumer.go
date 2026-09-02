// Package character consumes EVENT_TOPIC_CHARACTER_STATUS and tears the
// character out of whatever trade room they occupy (design §3.3).
//
// Three of the table's rows land here, and they do NOT share a leaveReason: a
// logout is a plain cancellation, while a map or channel change has its own
// client string ("the other person is on a different map"). The reason a trigger
// implies is decided here and handed to trade.TeardownCharacter — the processor
// never guesses it.
//
// Every handler discriminates on the envelope's `type` BEFORE its body is used:
// the topic carries every character-status family, and the bodies are structurally
// similar enough that an unfiltered handler would decode a LOGIN as a LOGOUT and
// tear down a live room.
//
// A room already in SETTLING is left alone; that refusal lives in
// trade.TeardownCharacter (FR-6.5), which claims the room through a state-checked
// removal rather than a read-then-remove.
package character

import (
	consumer2 "atlas-trades/kafka/consumer"
	characterKafka "atlas-trades/kafka/message/character"
	"atlas-trades/trade"
	"context"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("trade_character_status")(characterKafka.EnvEventTopicCharacterStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(rf func(topic string, handler handler.Handler) (string, error)) error {
			var t string
			var err error
			t, err = topic.EnvProvider(l)(characterKafka.EnvEventTopicCharacterStatus)()
			if err != nil {
				return err
			}
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

// handleStatusEventLogout ends the trade of a character who logged out or
// dropped their connection, telling the survivor TRADE_CANCELLED (design §3.3).
//
// Escrow recovery does not depend on the disconnecting client being reachable
// (FR-6.4): under the reserve-at-staging model nothing left either inventory, so
// the teardown's reservation cancels are the whole of the recovery.
func handleStatusEventLogout(db *gorm.DB) message.Handler[characterKafka.StatusEvent[characterKafka.StatusEventLogoutBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e characterKafka.StatusEvent[characterKafka.StatusEventLogoutBody]) {
		if e.Type != characterKafka.EventCharacterStatusTypeLogout {
			return
		}
		l.Debugf("Character [%d] has logged out. worldId [%d] channelId [%d] mapId [%d]. Tearing down their trade room.", e.CharacterId, e.WorldId, e.Body.ChannelId, e.Body.MapId)
		if err := trade.NewProcessor(l, ctx, db).TeardownCharacter(e.TransactionId, character.Id(e.CharacterId), trade.ReasonTradeCancelled); err != nil {
			l.WithError(err).Errorf("Unable to tear down the trade room of character [%d] on logout.", e.CharacterId)
		}
	}
}

// handleStatusEventMapChanged ends the trade of a character who left the map it
// was opened in. The reason is TRADE_DIFFERENT_MAP rather than TRADE_CANCELLED:
// the client has a distinct string for it, and a trade is field-scoped, so the
// map transition is itself the cause the players should be told about.
func handleStatusEventMapChanged(db *gorm.DB) message.Handler[characterKafka.StatusEvent[characterKafka.StatusEventMapChangedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e characterKafka.StatusEvent[characterKafka.StatusEventMapChangedBody]) {
		if e.Type != characterKafka.EventCharacterStatusTypeMapChanged {
			return
		}
		l.Debugf("Character [%d] has changed maps. worldId [%d] channelId [%d] oldMapId [%d] newMapId [%d]. Tearing down their trade room.", e.CharacterId, e.WorldId, e.Body.ChannelId, e.Body.OldMapId, e.Body.TargetMapId)
		if err := trade.NewProcessor(l, ctx, db).TeardownCharacter(e.TransactionId, character.Id(e.CharacterId), trade.ReasonTradeDifferentMap); err != nil {
			l.WithError(err).Errorf("Unable to tear down the trade room of character [%d] on map change.", e.CharacterId)
		}
	}
}

// handleStatusEventChannelChanged ends the trade of a character who switched
// channels. A channel change emits neither LOGOUT nor MAP_CHANGED, so without
// this arm the registry's member index would keep the character bound to a room
// on a channel they have left: every later create would fail ErrOwnerHasRoom and
// every later enter would answer OTHER_REQUESTS, indefinitely.
func handleStatusEventChannelChanged(db *gorm.DB) message.Handler[characterKafka.StatusEvent[characterKafka.ChangeChannelEventLoginBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e characterKafka.StatusEvent[characterKafka.ChangeChannelEventLoginBody]) {
		if e.Type != characterKafka.EventCharacterStatusTypeChannelChanged {
			return
		}
		l.Debugf("Character [%d] has changed channels. worldId [%d] channelId [%d] oldChannelId [%d]. Tearing down their trade room.", e.CharacterId, e.WorldId, e.Body.ChannelId, e.Body.OldChannelId)
		if err := trade.NewProcessor(l, ctx, db).TeardownCharacter(e.TransactionId, character.Id(e.CharacterId), trade.ReasonTradeDifferentMap); err != nil {
			l.WithError(err).Errorf("Unable to tear down the trade room of character [%d] on channel change.", e.CharacterId)
		}
	}
}
