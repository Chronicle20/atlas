package summon

import (
	consumer2 "atlas-channel/kafka/consumer"
	summon2 "atlas-channel/kafka/message/summon"
	"atlas-channel/listener"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	model2 "github.com/Chronicle20/atlas/libs/atlas-model/model"
	summonpkt "github.com/Chronicle20/atlas/libs/atlas-packet/summon/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("summon_status_event")(summon2.EnvEventTopicSummonStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(summon2.EnvEventTopicSummonStatus)()
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
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventAttacked(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDamaged(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

func handleStatusEventCreated(sc server.Model, wp writer.Producer) message.Handler[summon2.StatusEvent[summon2.StatusEventCreatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e summon2.StatusEvent[summon2.StatusEventCreatedBody]) {
		if e.Type != summon2.EventSummonStatusCreated {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance),
			session.Announce(l)(ctx)(wp)(summonpkt.SummonSpawnWriter)(
				writer.SummonSpawnBody(e.OwnerCharacterId, e.SummonId, e.SkillId, e.Body.SkillLevel, e.Body.X, e.Body.Y, e.Body.Stance, e.Body.MovementType, e.Body.Puppet, e.Body.Animated)))
		if err != nil {
			l.WithError(err).Errorf("Unable to spawn summon [%d] for characters in map [%d].", e.SummonId, e.MapId)
		}
	}
}

func handleStatusEventMoved(sc server.Model, wp writer.Producer) message.Handler[summon2.StatusEvent[summon2.StatusEventMovedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e summon2.StatusEvent[summon2.StatusEventMovedBody]) {
		if e.Type != summon2.EventSummonStatusMoved {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		// Broadcast to OTHER sessions only: the owner's client already rendered
		// the move locally, so re-sending would double-apply it.
		err := _map.NewProcessor(l, ctx).ForOtherSessionsInMap(sc.Field(e.MapId, e.Instance), e.OwnerCharacterId,
			session.Announce(l)(ctx)(wp)(summonpkt.SummonMoveWriter)(
				writer.SummonMoveBody(e.OwnerCharacterId, e.SummonId, e.Body.X, e.Body.Y, e.Body.RawMovement)))
		if err != nil {
			l.WithError(err).Errorf("Unable to move summon [%d] for characters in map [%d].", e.SummonId, e.MapId)
		}
	}
}

func handleStatusEventDestroyed(sc server.Model, wp writer.Producer) message.Handler[summon2.StatusEvent[summon2.StatusEventDestroyedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e summon2.StatusEvent[summon2.StatusEventDestroyedBody]) {
		if e.Type != summon2.EventSummonStatusDestroyed {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance),
			session.Announce(l)(ctx)(wp)(summonpkt.SummonRemoveWriter)(
				writer.SummonRemoveBody(e.OwnerCharacterId, e.SummonId, e.Body.Animated)))
		if err != nil {
			l.WithError(err).Errorf("Unable to remove summon [%d] for characters in map [%d].", e.SummonId, e.MapId)
		}
	}
}

func handleStatusEventAttacked(sc server.Model, wp writer.Producer) message.Handler[summon2.StatusEvent[summon2.StatusEventAttackedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e summon2.StatusEvent[summon2.StatusEventAttackedBody]) {
		if e.Type != summon2.EventSummonStatusAttacked {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		targets := make([]summonpkt.SummonAttackTarget, 0, len(e.Body.Targets))
		for _, t := range e.Body.Targets {
			targets = append(targets, summonpkt.NewSummonAttackTarget(t.MonsterId, t.Damage))
		}

		// Broadcast to OTHER sessions only: the owner's client already rendered
		// the attack locally, so re-sending would double-apply it.
		err := _map.NewProcessor(l, ctx).ForOtherSessionsInMap(sc.Field(e.MapId, e.Instance), e.OwnerCharacterId,
			session.Announce(l)(ctx)(wp)(summonpkt.SummonAttackWriter)(
				writer.SummonAttackBody(e.OwnerCharacterId, e.SummonId, e.Body.Direction, targets)))
		if err != nil {
			l.WithError(err).Errorf("Unable to broadcast summon [%d] attack for characters in map [%d].", e.SummonId, e.MapId)
		}
	}
}

func handleStatusEventDamaged(sc server.Model, wp writer.Producer) message.Handler[summon2.StatusEvent[summon2.StatusEventDamagedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e summon2.StatusEvent[summon2.StatusEventDamagedBody]) {
		if e.Type != summon2.EventSummonStatusDamaged {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		// Broadcast to OTHER sessions only: the owner's client already rendered
		// the damage locally, so re-sending would double-apply it. The DESTROYED
		// event (handled separately) broadcasts the SummonRemove when HP hits zero.
		err := _map.NewProcessor(l, ctx).ForOtherSessionsInMap(sc.Field(e.MapId, e.Instance), e.OwnerCharacterId,
			session.Announce(l)(ctx)(wp)(summonpkt.SummonDamageWriter)(
				writer.SummonDamageBody(e.OwnerCharacterId, e.SummonId, e.Body.Damage, e.Body.MonsterIdFrom)))
		if err != nil {
			l.WithError(err).Errorf("Unable to broadcast summon [%d] damage for characters in map [%d].", e.SummonId, e.MapId)
		}
	}
}
