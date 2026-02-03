package monster

import (
	consumer2 "atlas-channel/kafka/consumer"
	monster2 "atlas-channel/kafka/message/monster"
	_map "atlas-channel/map"
	"atlas-channel/monster"
	"atlas-channel/party"
	"atlas-channel/server"
	"atlas-channel/session"
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
	"github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("monster_status_event")(monster2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) {
				var t string
				t, _ = topic.EnvProvider(l)(monster2.EnvEventTopicStatus)()
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCreated(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDestroyed(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDamaged(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventKilled(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventStartControl(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventStopControl(sc, wp))))
			}
		}
	}
}

func handleStatusEventCreated(sc server.Model, wp writer.Producer) message.Handler[monster2.StatusEvent[monster2.StatusEventCreatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monster2.StatusEvent[monster2.StatusEventCreatedBody]) {
		if e.Type != monster2.EventStatusCreated {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		m, err := monster.NewProcessor(l, ctx).GetById(e.UniqueId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve the monster [%d] being spawned.", e.UniqueId)
			return
		}

		err = _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(_map2.Id(e.MapId), e.Instance), spawnForSession(l)(ctx)(wp)(m))
		if err != nil {
			l.WithError(err).Errorf("Unable to spawn monster [%d] for characters in map [%d].", m.UniqueId(), e.MapId)
		}
	}
}

func spawnForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
		return func(wp writer.Producer) func(m monster.Model) model.Operator[session.Model] {
			return func(m monster.Model) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.SpawnMonster)(writer.SpawnMonsterBody(l, tenant.MustFromContext(ctx))(m, true))
			}
		}
	}
}

func handleStatusEventDestroyed(sc server.Model, wp writer.Producer) message.Handler[monster2.StatusEvent[monster2.StatusEventDestroyedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monster2.StatusEvent[monster2.StatusEventDestroyedBody]) {
		if e.Type != monster2.EventStatusDestroyed {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(_map2.Id(e.MapId), e.Instance), destroyForSession(l)(ctx)(wp)(e.UniqueId))
		if err != nil {
			l.WithError(err).Errorf("Unable to destroy monster [%d] for characters in map [%d].", e.UniqueId, e.MapId)
		}
	}
}

func destroyForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(uniqueId uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(uniqueId uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(uniqueId uint32) model.Operator[session.Model] {
			return func(uniqueId uint32) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.DestroyMonster)(writer.DestroyMonsterBody(l, tenant.MustFromContext(ctx))(uniqueId, writer.DestroyMonsterTypeFadeOut))
			}
		}
	}
}

func handleStatusEventDamaged(sc server.Model, wp writer.Producer) message.Handler[monster2.StatusEvent[monster2.StatusEventDamagedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monster2.StatusEvent[monster2.StatusEventDamagedBody]) {
		if e.Type != monster2.EventStatusDamaged {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		m, err := monster.NewProcessor(l, ctx).GetById(e.UniqueId)
		if err != nil {
			return
		}

		var idProvider = model.FixedProvider([]uint32{e.Body.ActorId})

		p, err := party.NewProcessor(l, ctx).GetByMemberId(e.Body.ActorId)
		if err == nil {
			mimf := party.MemberInMap(sc.WorldId(), sc.ChannelId(), _map2.Id(e.MapId))
			mp := party.FilteredMemberProvider(mimf)(model.FixedProvider(p))
			idProvider = party.MemberToMemberIdMapper(mp)
		}

		err = session.NewProcessor(l, ctx).ForEachByCharacterId(sc.WorldId(), sc.ChannelId())(idProvider, session.Announce(l)(ctx)(wp)(writer.MonsterHealth)(writer.MonsterHealthBody(m)))
		if err != nil {
			l.WithError(err).Errorf("Unable to announce monster [%d] health.", e.UniqueId)
		}
	}
}

func handleStatusEventKilled(sc server.Model, wp writer.Producer) message.Handler[monster2.StatusEvent[monster2.StatusEventKilledBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monster2.StatusEvent[monster2.StatusEventKilledBody]) {
		if e.Type != monster2.EventStatusKilled {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(_map2.Id(e.MapId), e.Instance), killForSession(l)(ctx)(wp)(e.UniqueId))
		if err != nil {
			l.WithError(err).Errorf("Unable to kill monster [%d] for characters in map [%d].", e.UniqueId, e.MapId)
		}
	}
}

func killForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(uniqueId uint32) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(uniqueId uint32) model.Operator[session.Model] {
		return func(wp writer.Producer) func(uniqueId uint32) model.Operator[session.Model] {
			return func(uniqueId uint32) model.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(writer.DestroyMonster)(writer.DestroyMonsterBody(l, tenant.MustFromContext(ctx))(uniqueId, writer.DestroyMonsterTypeFadeOut))
			}
		}
	}
}

func handleStatusEventStartControl(sc server.Model, wp writer.Producer) message.Handler[monster2.StatusEvent[monster2.StatusEventStartControlBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monster2.StatusEvent[monster2.StatusEventStartControlBody]) {
		if e.Type != monster2.EventStatusStartControl {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		m := monster.NewModelBuilder(e.UniqueId, world.Id(e.WorldId), channel.Id(e.ChannelId), _map2.Id(e.MapId), e.MonsterId).
			SetControlCharacterId(e.Body.ActorId).
			SetX(e.Body.X).SetY(e.Body.Y).
			SetStance(e.Body.Stance).
			SetFH(e.Body.FH).
			SetTeam(e.Body.Team).
			MustBuild()
		sf := session.Announce(l)(ctx)(wp)(writer.ControlMonster)(writer.StartControlMonsterBody(l, tenant.MustFromContext(ctx))(m, false))
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.Body.ActorId, sf)
		if err != nil {
			l.WithError(err).Errorf("Unable to start control of monster [%d] for character [%d].", e.UniqueId, e.Body.ActorId)
		}
	}
}

func handleStatusEventStopControl(sc server.Model, wp writer.Producer) message.Handler[monster2.StatusEvent[monster2.StatusEventStopControlBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monster2.StatusEvent[monster2.StatusEventStopControlBody]) {
		if e.Type != monster2.EventStatusStopControl {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), world.Id(e.WorldId), channel.Id(e.ChannelId)) {
			return
		}

		m := monster.NewModelBuilder(e.UniqueId, world.Id(e.WorldId), channel.Id(e.ChannelId), _map2.Id(e.MapId), e.MonsterId).
			MustBuild()
		sf := session.Announce(l)(ctx)(wp)(writer.ControlMonster)(writer.StopControlMonsterBody(l, tenant.MustFromContext(ctx))(m))
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.WorldId(), sc.ChannelId())(e.Body.ActorId, sf)
		if err != nil {
			l.WithError(err).Errorf("Unable to stop control of monster [%d] for character [%d].", e.UniqueId, e.Body.ActorId)
		}
	}
}
