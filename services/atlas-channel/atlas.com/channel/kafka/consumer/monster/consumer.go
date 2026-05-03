package monster

import (
	"atlas-channel/character"
	skill2 "atlas-channel/character/skill"
	consumer2 "atlas-channel/kafka/consumer"
	monster2 "atlas-channel/kafka/message/monster"
	_map "atlas-channel/map"
	"atlas-channel/monster"
	"atlas-channel/party"
	"atlas-channel/server"
	"atlas-channel/session"
	socketHandler "atlas-channel/socket/handler"
	"atlas-channel/socket/writer"
	"context"
	"math"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	model2 "github.com/Chronicle20/atlas/libs/atlas-model/model"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
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

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) error {
			return func(rf func(topic string, handler handler.Handler) (string, error)) error {
				var t string
				t, _ = topic.EnvProvider(l)(monster2.EnvEventTopicStatus)()
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCreated(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDestroyed(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDamaged(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventKilled(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventStartControl(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventStopControl(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventAggroChanged(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEffectApplied(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEffectExpired(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEffectCancelled(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleDamageReflected(sc)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventNextSkillDecided(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMpChanged(sc, wp)))); err != nil {
					return err
				}
				return nil
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

		// Send the initial Control packet in the same goroutine, immediately
		// after Spawn, so the v83 client always sees Spawn-then-Control for
		// fresh mobs. atlas-monsters' Create() now assigns the controller in
		// Redis without emitting a StartControl event, deferring the wire
		// notification to here. This eliminates the parallel-handler race
		// (atlas-kafka manager.go:437 spawns one goroutine per registered
		// handler) that previously let Control land before Spawn and caused
		// slope-spawn fall-throughs.
		if m.ControlCharacterId() != 0 {
			sf := session.Announce(l)(ctx)(wp)(monsterpkt.MonsterControlWriter)(writer.StartControlMonsterBody(m, m.ControllerHasAggro()))
			if cerr := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(m.ControlCharacterId(), sf); cerr != nil {
				l.WithError(cerr).Errorf("Unable to send initial control of monster [%d] to character [%d].", m.UniqueId(), m.ControlCharacterId())
			}
		}
	}
}

func spawnForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(m monster.Model) model2.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(m monster.Model) model2.Operator[session.Model] {
		return func(wp writer.Producer) func(m monster.Model) model2.Operator[session.Model] {
			return func(m monster.Model) model2.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(monsterpkt.MonsterSpawnWriter)(writer.SpawnMonsterBody(m, true))
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
		t := tenant.MustFromContext(ctx)
		monster.GetNextSkillInbox().Evict(t, e.UniqueId)
		monster.GetStatusMirror().OnMonsterGone(t, e.UniqueId)
	}
}

func destroyForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(uniqueId uint32) model2.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(uniqueId uint32) model2.Operator[session.Model] {
		return func(wp writer.Producer) func(uniqueId uint32) model2.Operator[session.Model] {
			return func(uniqueId uint32) model2.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(monsterpkt.MonsterDestroyWriter)(monsterpkt.NewMonsterDestroy(uniqueId, monsterpkt.DestroyTypeFadeOut).Encode)
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

		hpPercent := byte(math.Max(1, float64(m.Hp())*100/float64(m.MaxHp())))
		announcer := session.Announce(l)(ctx)(wp)(monsterpkt.MonsterHealthWriter)(monsterpkt.NewMonsterHealth(m.UniqueId(), hpPercent).Encode)

		// Boss monsters: broadcast HP bar to all characters in the map
		f := sc.Field(e.MapId, e.Instance)
		go func() {
			if e.Body.Boss {
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
		}()
		// Only echo a MonsterDamage packet for damage sources that have no
		// corresponding client-side attack broadcast. Player attacks are
		// already rendered to observers by CharacterAttack*Writer in
		// socket/handler/character_attack_common.go, so emitting here too
		// would double-render the damage number.
		if e.Body.DamageSource == monster2.DamageSourceMonsterAttack || e.Body.DamageSource == monster2.DamageSourceDamageOverTime {
			go func() {
				err = _map.NewProcessor(l, ctx).ForSessionsInMap(f, func(s session.Model) error {
					de := e.Body.DamageEntries[len(e.Body.DamageEntries)-1]
					return session.Announce(l)(ctx)(wp)(monsterpkt.MonsterDamageWriter)(monsterpkt.NewMonsterDamage(m.UniqueId(), monsterpkt.MonsterDamageTypeUnk3, uint32(de.Damage), m.Hp(), m.MaxHp()).Encode)(s)
				})
			}()
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
		monster.GetStatusMirror().OnMonsterGone(tenant.MustFromContext(ctx), e.UniqueId)
	}
}

func killForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(uniqueId uint32) model2.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(uniqueId uint32) model2.Operator[session.Model] {
		return func(wp writer.Producer) func(uniqueId uint32) model2.Operator[session.Model] {
			return func(uniqueId uint32) model2.Operator[session.Model] {
				return session.Announce(l)(ctx)(wp)(monsterpkt.MonsterDestroyWriter)(monsterpkt.NewMonsterDestroy(uniqueId, monsterpkt.DestroyTypeFadeOut).Encode)
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
		sf := session.Announce(l)(ctx)(wp)(monsterpkt.MonsterControlWriter)(writer.StartControlMonsterBody(m, e.Body.ControllerHasAggro))
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
		sf := session.Announce(l)(ctx)(wp)(monsterpkt.MonsterControlWriter)(writer.StopControlMonsterBody(m))
		err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.Body.ActorId, sf)
		if err != nil {
			l.WithError(err).Errorf("Unable to stop control of monster [%d] for character [%d].", e.UniqueId, e.Body.ActorId)
		}
	}
}

func handleStatusEventAggroChanged(sc server.Model, wp writer.Producer) message.Handler[monster2.StatusEvent[monster2.StatusEventAggroChangedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monster2.StatusEvent[monster2.StatusEventAggroChangedBody]) {
		if e.Type != monster2.EventStatusAggroChanged {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		m, err := monster.NewProcessor(l, ctx).GetById(e.UniqueId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve monster [%d] for aggro change.", e.UniqueId)
			return
		}
		sf := session.Announce(l)(ctx)(wp)(monsterpkt.MonsterControlWriter)(writer.StartControlMonsterBody(m, e.Body.ControllerHasAggro))
		err = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.Body.ControllerCharacterId, sf)
		if err != nil {
			l.WithError(err).Errorf("Unable to refresh control state for monster [%d] for character [%d].", e.UniqueId, e.Body.ControllerCharacterId)
		}
	}
}

// statusVenomKey is the Statuses map key used by atlas-monsters to denote
// the VENOM stat. Centralised here so the wire-collapse gate has a single
// source of truth.
const statusVenomKey = "VENOM"

// monsterStatBroadcaster is the channel-side broadcast seam. The handlers
// below build a *MonsterTemporaryStat and ask the broadcaster to fan it
// out to every session in the map. Held as package-level vars so tests
// can swap in a recording spy without standing up a REST mock for
// _map.ForSessionsInMap. The defaults preserve the historical behaviour
// of announcing through wp + session.Announce.
var monsterStatSetBroadcaster = func(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, f field.Model, uniqueId uint32, stat *packetmodel.MonsterTemporaryStat) {
	err := _map.NewProcessor(l, ctx).ForSessionsInMap(f,
		session.Announce(l)(ctx)(wp)(monsterpkt.MonsterStatSetWriter)(monsterpkt.NewMonsterStatSet(uniqueId, stat).Encode))
	if err != nil {
		l.WithError(err).Errorf("Unable to broadcast status effect applied to monster [%d].", uniqueId)
	}
}

var monsterStatResetBroadcaster = func(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, f field.Model, uniqueId uint32, stat *packetmodel.MonsterTemporaryStat) {
	err := _map.NewProcessor(l, ctx).ForSessionsInMap(f,
		session.Announce(l)(ctx)(wp)(monsterpkt.MonsterStatResetWriter)(monsterpkt.NewMonsterStatReset(uniqueId, stat).Encode))
	if err != nil {
		l.WithError(err).Errorf("Unable to broadcast status effect reset for monster [%d].", uniqueId)
	}
}

// statusesWithoutVenom returns a copy of statuses with VENOM removed.
// Callers use it to broadcast a non-VENOM-only stat-set/reset when VENOM
// is being collapsed.
func statusesWithoutVenom(in map[string]int32) map[string]int32 {
	if len(in) == 0 {
		return in
	}
	out := make(map[string]int32, len(in))
	for k, v := range in {
		if k == statusVenomKey {
			continue
		}
		out[k] = v
	}
	return out
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

		// Capture pre-OnApplied venom count so we can detect the
		// 0->1 transition. We only collapse subsequent applies on
		// the wire; the first (transition-to-active) still emits a
		// MonsterStatSet so the client renders the stat icon.
		_, isVenom := e.Body.Statuses[statusVenomKey]
		priorVenomCount := 0
		if isVenom {
			priorVenomCount = monster.GetStatusMirror().VenomCount(t, e.UniqueId)
		}

		// Update the mirror BEFORE the broadcast decision so that
		// downstream consumers (and follow-up logic) see post-apply
		// state. The transition decision uses the snapshot above.
		monster.GetStatusMirror().OnApplied(t, e.UniqueId, monster.StatusEffectAppliedBody{
			EffectId:          e.Body.EffectId,
			SourceType:        e.Body.SourceType,
			SourceCharacterId: e.Body.SourceCharacterId,
			SourceSkillId:     e.Body.SourceSkillId,
			SourceSkillLevel:  e.Body.SourceSkillLevel,
			Statuses:          e.Body.Statuses,
			Duration:          int64(e.Body.Duration),
			ReflectKind:       e.Body.ReflectKind,
			ReflectPercent:    e.Body.ReflectPercent,
			ReflectLtX:        e.Body.ReflectLtX,
			ReflectLtY:        e.Body.ReflectLtY,
			ReflectRbX:        e.Body.ReflectRbX,
			ReflectRbY:        e.Body.ReflectRbY,
			ReflectMaxDamage:  e.Body.ReflectMaxDamage,
		}, time.Now())

		// Wire-collapse: if VENOM is already active on this monster
		// before this apply, suppress the VENOM portion of the
		// broadcast. Non-VENOM statuses in the same body still
		// broadcast normally.
		statuses := e.Body.Statuses
		if isVenom && priorVenomCount > 0 {
			statuses = statusesWithoutVenom(e.Body.Statuses)
		}
		if len(statuses) == 0 {
			return
		}

		stat := packetmodel.NewMonsterTemporaryStat()
		for s, a := range statuses {
			stat.AddStat(l)(t)(s, e.Body.SourceSkillId, e.Body.SourceSkillLevel, a, time.Now().Add(time.Duration(e.Body.Duration)*time.Millisecond))
		}

		monsterStatSetBroadcaster(l, ctx, sc, wp, sc.Field(e.MapId, e.Instance), e.UniqueId, stat)
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

		// Update the mirror first so VenomCount reflects the
		// post-removal state used by the wire-collapse decision.
		monster.GetStatusMirror().OnExpired(t, e.UniqueId, e.Body.EffectId)

		_, isVenom := e.Body.Statuses[statusVenomKey]
		statuses := e.Body.Statuses
		if isVenom && monster.GetStatusMirror().VenomCount(t, e.UniqueId) > 0 {
			statuses = statusesWithoutVenom(e.Body.Statuses)
		}
		if len(statuses) == 0 {
			return
		}

		stat := packetmodel.NewMonsterTemporaryStat()
		for s, a := range statuses {
			stat.AddStat(l)(t)(s, 0, 0, a, time.Now())
		}

		monsterStatResetBroadcaster(l, ctx, sc, wp, sc.Field(e.MapId, e.Instance), e.UniqueId, stat)
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

		// Update the mirror first so VenomCount reflects the
		// post-removal state used by the wire-collapse decision.
		monster.GetStatusMirror().OnCancelled(t, e.UniqueId, e.Body.EffectId)

		_, isVenom := e.Body.Statuses[statusVenomKey]
		statuses := e.Body.Statuses
		if isVenom && monster.GetStatusMirror().VenomCount(t, e.UniqueId) > 0 {
			statuses = statusesWithoutVenom(e.Body.Statuses)
		}
		if len(statuses) == 0 {
			return
		}

		stat := packetmodel.NewMonsterTemporaryStat()
		for s, a := range statuses {
			stat.AddStat(l)(t)(s, 0, 0, a, time.Now())
		}

		monsterStatResetBroadcaster(l, ctx, sc, wp, sc.Field(e.MapId, e.Instance), e.UniqueId, stat)
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

func handleStatusEventNextSkillDecided(sc server.Model, _ writer.Producer) message.Handler[monster2.StatusEvent[monster2.StatusEventNextSkillDecidedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monster2.StatusEvent[monster2.StatusEventNextSkillDecidedBody]) {
		if e.Type != monster2.EventStatusNextSkillDecided {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		t := tenant.MustFromContext(ctx)
		monster.GetNextSkillInbox().Put(t, e.UniqueId, monster.Decision{
			SkillId:                e.Body.SkillId,
			SkillLevel:             e.Body.SkillLevel,
			DecidedAtMs:            e.Body.DecidedAtMs,
			NextEligibleRepickAtMs: e.Body.NextEligibleRepickAtMs,
		})
		l.Debugf("Inbox: stored decision (skill=%d level=%d) for monster [%d].", e.Body.SkillId, e.Body.SkillLevel, e.UniqueId)
	}
}

func handleStatusEventMpChanged(sc server.Model, wp writer.Producer) message.Handler[monster2.StatusEvent[monster2.StatusEventMpChangedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monster2.StatusEvent[monster2.StatusEventMpChangedBody]) {
		if e.Type != monster2.EventStatusMpChanged {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		s, err := session.NewProcessor(l, ctx).GetByCharacterId(sc.Channel())(e.Body.CharacterId)
		if err != nil {
			return
		}

		switch e.Body.Reason {
		case monster2.MpChangeReasonMpEater:
			var c character.Model
			cp := character.NewProcessor(l, ctx)
			c, err = cp.GetById(cp.SkillModelDecorator)(e.Body.CharacterId)
			if err != nil {
				l.WithError(err).Errorf("Unable to locate character [%d] causing MP change via skill.", e.Body.CharacterId)
				return
			}
			var sk skill2.Model
			sk, err = c.SkillById(skill.Id(e.Body.SkillId))
			if err != nil {
				l.WithError(err).Errorf("Unable to locate skill [%d]", e.Body.SkillId)
				return
			}

			f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
			if err = cp.ChangeMP(f, e.Body.CharacterId, int16(e.Body.Amount)); err != nil {
				l.WithError(err).Errorf("MP_CHANGED MP_EATER: ChangeMP failed for character [%d].", e.Body.CharacterId)
			}

			err = socketHandler.AnnounceSkillUse(l)(ctx)(wp)(e.Body.SkillId, c.Level(), sk.Level())(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to announce Skill Use: [%d].", e.Body.CharacterId)
			}
			_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(f, e.Body.CharacterId,
				socketHandler.AnnounceForeignSkillUse(l)(ctx)(wp)(e.Body.CharacterId, e.Body.SkillId, c.Level(), sk.Level()),
			)
		default:
			l.Debugf("MP_CHANGED: ignoring unknown reason [%s] for monster [%d].", e.Body.Reason, e.UniqueId)
		}
	}
}
