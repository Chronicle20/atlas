package dragon

import (
	consumer2 "atlas-channel/kafka/consumer"
	dragonmsg "atlas-channel/kafka/message/dragon"
	"atlas-channel/listener"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	model2 "github.com/Chronicle20/atlas/libs/atlas-model/model"
	dragonpkt "github.com/Chronicle20/atlas/libs/atlas-packet/dragon/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// excludesOwner reports whether the owner is excluded from this event's
// broadcast. MOVED excludes them: the owner's client already rendered the motion
// locally, so re-sending double-applies it — the same reasoning as the summon
// move relay. CREATED and DESTROYED go map-wide including the owner, because the
// owner has not rendered either locally.
func excludesOwner(eventType string) bool {
	return eventType == dragonmsg.EventDragonStatusMoved
}

// handles reports whether eventType is one this consumer broadcasts.
func handles(eventType string) bool {
	switch eventType {
	case dragonmsg.EventDragonStatusCreated,
		dragonmsg.EventDragonStatusMoved,
		dragonmsg.EventDragonStatusDestroyed:
		return true
	}
	return false
}

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("dragon_status_event")(dragonmsg.EnvEventTopicDragonStatus)(consumerGroupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser),
				consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var handles []listener.HandlerHandle
				t, _ := topic.EnvProvider(l)(dragonmsg.EnvEventTopicDragonStatus)()

				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCreated(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})

				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMoved(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})

				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDestroyed(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

func handleStatusEventCreated(sc server.Model, wp writer.Producer) message.Handler[dragonmsg.StatusEvent[dragonmsg.StatusEventCreatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e dragonmsg.StatusEvent[dragonmsg.StatusEventCreatedBody]) {
		if e.Type != dragonmsg.EventDragonStatusCreated {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		// Map-wide INCLUDING the owner (FR-3.1): the owner has not rendered its
		// own dragon locally.
		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance),
			session.Announce(l)(ctx)(wp)(dragonpkt.DragonSpawnWriter)(
				writer.DragonSpawnBody(e.OwnerCharacterId, e.Body.X, e.Body.Y, e.Body.Stance, e.Body.JobId)))
		if err != nil {
			l.WithError(err).Errorf("Unable to spawn dragon for character [%d] in map [%d].", e.OwnerCharacterId, e.MapId)
		}
	}
}

func handleStatusEventMoved(sc server.Model, wp writer.Producer) message.Handler[dragonmsg.StatusEvent[dragonmsg.StatusEventMovedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e dragonmsg.StatusEvent[dragonmsg.StatusEventMovedBody]) {
		if e.Type != dragonmsg.EventDragonStatusMoved {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		// OTHER sessions only (FR-4.3): the owner's client already rendered the
		// motion locally, so re-sending would double-apply it.
		err := _map.NewProcessor(l, ctx).ForOtherSessionsInMap(sc.Field(e.MapId, e.Instance), e.OwnerCharacterId,
			session.Announce(l)(ctx)(wp)(dragonpkt.DragonMoveWriter)(
				writer.DragonMoveBody(e.OwnerCharacterId, e.Body.RawMovement)))
		if err != nil {
			l.WithError(err).Errorf("Unable to move dragon for character [%d] in map [%d].", e.OwnerCharacterId, e.MapId)
		}
	}
}

func handleStatusEventDestroyed(sc server.Model, wp writer.Producer) message.Handler[dragonmsg.StatusEvent[dragonmsg.StatusEventDestroyedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e dragonmsg.StatusEvent[dragonmsg.StatusEventDestroyedBody]) {
		if e.Type != dragonmsg.EventDragonStatusDestroyed {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		// Map-wide (FR-3.3). Note the client discards REMOVE_DRAGON — it has no
		// handler arm for the opcode. The dragon actually disappears because the
		// owner's CUser is destroyed when they leave the field, which the
		// character-removal path already does. This broadcast is correct to send
		// and is not the mechanism.
		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance),
			session.Announce(l)(ctx)(wp)(dragonpkt.DragonRemoveWriter)(
				writer.DragonRemoveBody(e.OwnerCharacterId)))
		if err != nil {
			l.WithError(err).Errorf("Unable to remove dragon for character [%d] in map [%d].", e.OwnerCharacterId, e.MapId)
		}
	}
}
