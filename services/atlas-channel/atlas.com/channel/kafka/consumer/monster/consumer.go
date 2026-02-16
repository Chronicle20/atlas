package monster

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	monster2 "atlas-channel/kafka/message/monster"
	_map "atlas-channel/map"
	"atlas-channel/monster"
	"atlas-channel/party"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas-kafka/handler"
	"github.com/Chronicle20/atlas-kafka/message"
	"github.com/Chronicle20/atlas-kafka/topic"
	model2 "github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
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
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEffectApplied(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEffectExpired(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEffectCancelled(sc, wp))))
				_, _ = rf(t, message.AdaptHandler(message.PersistentConfig(handleDamageReflected(sc))))
			}
		}
	}
}

func handleStatusEventCreated(sc server.Model, wp writer.Producer) message.Handler[monster2.StatusEvent[monster2.StatusEventCreatedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monster2.StatusEvent[monster2.StatusEventCreatedBody]) {
		if e.Type != monster2.EventStatusCreated {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		m, err := monster.NewProcessor(l, ctx).GetById(e.UniqueId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve the monster [%d] being spawned.", e.UniqueId)
			return
		}

		err = _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance), spawnForSession(l)(ctx)(wp)(m))
		if err != nil {
			l.WithError(err).Errorf("Unable to spawn monster [%d] for characters in map [%d].", m.UniqueId(), e.MapId)
		}
	}
}

func spawnForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(m monster.Model) model2.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(m monster.Model) model2.Operator[session.Model] {
		return func(wp writer.Producer) func(m monster.Model) model2.Operator[session.Model] {
			return func(m monster.Model) model2.Operator[session.Model] {
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

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance), destroyForSession(l)(ctx)(wp)(e.UniqueId))
		if err != nil {
			l.WithError(err).Errorf("Unable to destroy monster [%d] for characters in map [%d].", e.UniqueId, e.MapId)
		}
	}
}

func destroyForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(uniqueId uint32) model2.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(uniqueId uint32) model2.Operator[session.Model] {
		return func(wp writer.Producer) func(uniqueId uint32) model2.Operator[session.Model] {
			return func(uniqueId uint32) model2.Operator[session.Model] {
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

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		m, err := monster.NewProcessor(l, ctx).GetById(e.UniqueId)
		if err != nil {
			return
		}

		announcer := session.Announce(l)(ctx)(wp)(writer.MonsterHealth)(writer.MonsterHealthBody(m))

		// Boss monsters: broadcast HP bar to all characters in the map
		if e.Body.Boss {
			f := sc.Field(e.MapId, e.Instance)
			err = _map.NewProcessor(l, ctx).ForSessionsInMap(f, announcer)
		} else {
			var idProvider = model2.FixedProvider([]uint32{e.Body.ActorId})

			p, err2 := party.NewProcessor(l, ctx).GetByMemberId(e.Body.ActorId)
			if err2 == nil {
				f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
				mimf := party.MemberInMap(f)
				mp := party.FilteredMemberProvider(mimf)(model2.FixedProvider(p))
				idProvider = party.MemberToMemberIdMapper(mp)
			}

			err = session.NewProcessor(l, ctx).ForEachByCharacterId(sc.Channel())(idProvider, announcer)
		}
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

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance), killForSession(l)(ctx)(wp)(e.UniqueId))
		if err != nil {
			l.WithError(err).Errorf("Unable to kill monster [%d] for characters in map [%d].", e.UniqueId, e.MapId)
		}
	}
}

func killForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(uniqueId uint32) model2.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(uniqueId uint32) model2.Operator[session.Model] {
		return func(wp writer.Producer) func(uniqueId uint32) model2.Operator[session.Model] {
			return func(uniqueId uint32) model2.Operator[session.Model] {
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

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
		m := monster.NewModelBuilder(e.UniqueId, f, e.MonsterId).
			SetControlCharacterId(e.Body.ActorId).
			SetX(e.Body.X).SetY(e.Body.Y).
			SetStance(e.Body.Stance).
			SetFH(e.Body.FH).
			SetTeam(e.Body.Team).
			MustBuild()
		sf := session.Announce(l)(ctx)(wp)(writer.ControlMonster)(writer.StartControlMonsterBody(l, tenant.MustFromContext(ctx))(m, false))
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.Body.ActorId, sf)
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

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
		m := monster.NewModelBuilder(e.UniqueId, f, e.MonsterId).
			MustBuild()
		sf := session.Announce(l)(ctx)(wp)(writer.ControlMonster)(writer.StopControlMonsterBody(l, tenant.MustFromContext(ctx))(m))
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.Body.ActorId, sf)
		if err != nil {
			l.WithError(err).Errorf("Unable to stop control of monster [%d] for character [%d].", e.UniqueId, e.Body.ActorId)
		}
	}
}

func handleStatusEffectApplied(sc server.Model, wp writer.Producer) message.Handler[monster2.StatusEvent[monster2.StatusEffectAppliedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monster2.StatusEvent[monster2.StatusEffectAppliedBody]) {
		if e.Type != monster2.EventStatusEffectApplied {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		t := tenant.MustFromContext(ctx)
		stat := model.NewMonsterTemporaryStat()
		for s, a := range e.Body.Statuses {
			stat.AddStat(l)(t)(s, e.Body.SourceSkillId, e.Body.SourceSkillLevel, a, time.Now().Add(time.Duration(e.Body.Duration)*time.Millisecond))
		}

		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance),
			session.Announce(l)(ctx)(wp)(writer.MonsterStatSet)(writer.MonsterStatSetBody(l, t)(e.UniqueId, stat)))
		if err != nil {
			l.WithError(err).Errorf("Unable to broadcast status effect applied to monster [%d].", e.UniqueId)
		}
	}
}

func handleStatusEffectExpired(sc server.Model, wp writer.Producer) message.Handler[monster2.StatusEvent[monster2.StatusEffectExpiredBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monster2.StatusEvent[monster2.StatusEffectExpiredBody]) {
		if e.Type != monster2.EventStatusEffectExpired {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		t := tenant.MustFromContext(ctx)
		stat := model.NewMonsterTemporaryStat()
		for s, a := range e.Body.Statuses {
			stat.AddStat(l)(t)(s, 0, 0, a, time.Now())
		}

		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance),
			session.Announce(l)(ctx)(wp)(writer.MonsterStatReset)(writer.MonsterStatResetBody(l, t)(e.UniqueId, stat)))
		if err != nil {
			l.WithError(err).Errorf("Unable to broadcast status effect expired from monster [%d].", e.UniqueId)
		}
	}
}

func handleStatusEffectCancelled(sc server.Model, wp writer.Producer) message.Handler[monster2.StatusEvent[monster2.StatusEffectCancelledBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monster2.StatusEvent[monster2.StatusEffectCancelledBody]) {
		if e.Type != monster2.EventStatusEffectCancelled {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		t := tenant.MustFromContext(ctx)
		stat := model.NewMonsterTemporaryStat()
		for s, a := range e.Body.Statuses {
			stat.AddStat(l)(t)(s, 0, 0, a, time.Now())
		}

		err := _map.NewProcessor(l, ctx).ForSessionsInMap(sc.Field(e.MapId, e.Instance),
			session.Announce(l)(ctx)(wp)(writer.MonsterStatReset)(writer.MonsterStatResetBody(l, t)(e.UniqueId, stat)))
		if err != nil {
			l.WithError(err).Errorf("Unable to broadcast status effect cancelled from monster [%d].", e.UniqueId)
		}
	}
}

func handleDamageReflected(sc server.Model) message.Handler[monster2.StatusEvent[monster2.StatusEventDamageReflectedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monster2.StatusEvent[monster2.StatusEventDamageReflectedBody]) {
		if e.Type != monster2.EventStatusDamageReflected {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
		_ = character.NewProcessor(l, ctx).ChangeHP(f, e.Body.CharacterId, -int16(e.Body.ReflectDamage))
	}
}
