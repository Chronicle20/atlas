package party

import (
	consumer2 "atlas-doors/kafka/consumer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
)

// The door service no longer derives door visibility from party membership.
//
// The Mystic Door's area door is a plain ranged map object — shown to every
// session in the map, like a monster. Party membership only gates door ENTRY
// and the town-portal array. That town-portal/party rendering is owned entirely
// by the channel's party-status consumer, which rebuilds the PARTYDATA
// town-portal array on every membership change.
//
// Re-deriving door scope here (the former ReconcileParty) re-broadcast the area
// door on every party change; the v83 client treats a repeat area-door spawn as
// an open/close toggle, which corrupted the render (door drawn below the
// platform) and cleared the door's frame layer — the source of the door-removal
// (expiry) crash. So these handlers are intentionally no-ops; the consumer stays
// registered only to drain the topic.

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("party_status_event")(EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		t, _ := topic.EnvProvider(l)(EnvEventStatusTopic)()
		handlers := []handler.Handler{
			message.AdaptHandler(message.PersistentConfig(handleJoined(l))),
			message.AdaptHandler(message.PersistentConfig(handleLeft(l))),
			message.AdaptHandler(message.PersistentConfig(handleExpel(l))),
			message.AdaptHandler(message.PersistentConfig(handleDisband(l))),
			message.AdaptHandler(message.PersistentConfig(handleChangeLeader(l))),
		}
		for _, h := range handlers {
			if _, err := rf(t, h); err != nil {
				return err
			}
		}
		return nil
	}
}

// The handlers below are intentionally no-ops (see the package note). They exist
// only so the consumer drains the party-status topic; door rendering no longer
// reacts to party membership.

func handleJoined(_ logrus.FieldLogger) message.Handler[StatusEvent[JoinedEventBody]] {
	return func(_ logrus.FieldLogger, _ context.Context, _ StatusEvent[JoinedEventBody]) {}
}

func handleLeft(_ logrus.FieldLogger) message.Handler[StatusEvent[LeftEventBody]] {
	return func(_ logrus.FieldLogger, _ context.Context, _ StatusEvent[LeftEventBody]) {}
}

func handleExpel(_ logrus.FieldLogger) message.Handler[StatusEvent[ExpelEventBody]] {
	return func(_ logrus.FieldLogger, _ context.Context, _ StatusEvent[ExpelEventBody]) {}
}

func handleDisband(_ logrus.FieldLogger) message.Handler[StatusEvent[DisbandEventBody]] {
	return func(_ logrus.FieldLogger, _ context.Context, _ StatusEvent[DisbandEventBody]) {}
}

func handleChangeLeader(_ logrus.FieldLogger) message.Handler[StatusEvent[ChangeLeaderEventBody]] {
	return func(_ logrus.FieldLogger, _ context.Context, _ StatusEvent[ChangeLeaderEventBody]) {}
}
