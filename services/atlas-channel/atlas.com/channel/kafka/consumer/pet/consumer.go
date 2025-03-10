package pet

import (
	consumer2 "atlas-channel/kafka/consumer"
	_map "atlas-channel/map"
	"atlas-channel/movement"
	"atlas-channel/pet"
	"atlas-channel/server"
	"atlas-channel/session"
	model2 "atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-constants/channel"
	_map2 "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("pet_status_event")(EnvStatusEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
			rf(consumer2.NewConfig(l)("pet_movement_event")(EnvEventTopicMovement)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(EnvStatusEventTopic)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleSpawned(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleDespawned(sc, wp))))
				t, _ = topic.EnvProvider(l)(EnvEventTopicMovement)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleMovementEvent(sc, wp))))
			}
		}
	}
}

func handleSpawned(sc server.Model, wp writer.Producer) message.Handler[statusEvent[spawnedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[spawnedStatusEventBody]) {
		if e.Type != StatusEventTypeSpawned {
			return
		}

		s, err := session.GetByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.OwnerId)
		if err != nil {
			return
		}

		p := pet.NewModelBuilder(e.PetId, 0, e.Body.TemplateId, e.Body.Name).
			SetSlot(e.Body.Slot).
			SetLevel(e.Body.Level).
			SetTameness(e.Body.Tameness).
			SetFullness(e.Body.Fullness).
			SetX(e.Body.X).
			SetY(e.Body.Y).
			SetStance(e.Body.Stance).
			SetFoothold(e.Body.FH).
			Build()

		go func() {
			_ = session.Announce(l)(ctx)(wp)(writer.PetActivated)(writer.PetSpawnBody(l)(sc.Tenant())(s.CharacterId(), p))(s)
			err = enableActions(l)(ctx)(wp)(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.StatChanged, s.CharacterId())
			}
		}()
		go func() {
			_ = _map.ForOtherSessionsInMap(l)(ctx)(s.Map(), s.CharacterId(), session.Announce(l)(ctx)(wp)(writer.PetActivated)(writer.PetSpawnBody(l)(sc.Tenant())(s.CharacterId(), p)))
		}()
	}
}

func handleDespawned(sc server.Model, wp writer.Producer) message.Handler[statusEvent[despawnedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e statusEvent[despawnedStatusEventBody]) {
		if e.Type != StatusEventTypeDespawned {
			return
		}

		s, err := session.GetByCharacterId(sc.Tenant(), sc.WorldId(), sc.ChannelId())(e.OwnerId)
		if err != nil {
			return
		}

		go func() {
			_ = session.Announce(l)(ctx)(wp)(writer.PetActivated)(writer.PetDespawnBody(s.CharacterId(), e.Body.Slot, writer.PetDespawnModeNormal))(s)
			err = enableActions(l)(ctx)(wp)(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write [%s] for character [%d].", writer.StatChanged, s.CharacterId())
			}
		}()
		go func() {
			_ = _map.ForOtherSessionsInMap(l)(ctx)(s.Map(), s.CharacterId(), session.Announce(l)(ctx)(wp)(writer.PetActivated)(writer.PetDespawnBody(s.CharacterId(), e.Body.Slot, writer.PetDespawnModeNormal)))
		}()
	}
}

func enableActions(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) error {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) error {
		return func(wp writer.Producer) func(s session.Model) error {
			return session.Announce(l)(ctx)(wp)(writer.StatChanged)(writer.StatChangedBody(l)(make([]model2.StatUpdate, 0), true))
		}
	}
}

func handleMovementEvent(sc server.Model, wp writer.Producer) message.Handler[movementEvent] {
	return func(l logrus.FieldLogger, ctx context.Context, e movementEvent) {
		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		p := pet.NewModelBuilder(e.PetId, 0, 0, "").
			SetOwnerID(e.OwnerId).
			SetSlot(e.Slot).
			Build()

		err := _map.ForOtherSessionsInMap(l)(ctx)(sc.Map(_map2.Id(e.MapId)), e.OwnerId, showMovementForSession(l)(ctx)(wp)(p, e))
		if err != nil {
			l.WithError(err).Errorf("Unable to move pet [%d] for characters in map [%d].", e.OwnerId, e.MapId)
		}
	}
}

func showMovementForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(p pet.Model, event movementEvent) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(p pet.Model, event movementEvent) model.Operator[session.Model] {
		return func(wp writer.Producer) func(p pet.Model, event movementEvent) model.Operator[session.Model] {
			return func(p pet.Model, event movementEvent) model.Operator[session.Model] {
				return func(s session.Model) error {
					l.Debugf("Writing pet [%d] movement for session [%s].", p.Id(), s.SessionId().String())

					mv := movement.ProduceMovementForSocket(event.Movement)
					return session.Announce(l)(ctx)(wp)(writer.PetMovement)(writer.PetMovementBody(l, tenant.MustFromContext(ctx))(p, *mv))(s)
				}
			}
		}
	}
}
