package monster

import (
	"atlas-channel/character"
	skill2 "atlas-channel/character/skill"
	consumer2 "atlas-channel/kafka/consumer"
	consumable2 "atlas-channel/kafka/message/consumable"
	monster2 "atlas-channel/kafka/message/monster"
	"atlas-channel/listener"
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

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	model2 "github.com/Chronicle20/atlas/libs/atlas-model/model"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	statpkt "github.com/Chronicle20/atlas/libs/atlas-packet/stat/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-rest/degrade"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model2.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("monster_status_event")(monster2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser, consumer.EnvHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(monster2.EnvEventTopicStatus)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCreated(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDestroyed(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDamaged(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventKilled(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventStartControl(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventStopControl(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventAggroChanged(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEffectApplied(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEffectExpired(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEffectCancelled(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleDamageReflected(sc))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventNextSkillDecided(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventMpChanged(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCaught(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventCatchFailed(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
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

		m, err := monsterGetByIdFn(l, ctx, e.UniqueId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve the monster [%d] being spawned.", e.UniqueId)
			return
		}

		// Seed the live mirror from the model already fetched for the spawn
		// packet (design §3 OQ1). An event landing between the REST read and
		// this Put has a millisecond window at spawn time (MP at max) and
		// self-corrects on the next MP/aggro event.
		monster.GetLiveMirror().Put(tenant.MustFromContext(ctx), e.UniqueId, monster.LiveEntryFromModel(m))

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
		monster.GetLiveMirror().Remove(t, e.UniqueId)
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

// shouldEchoDamagePacket reports whether a DAMAGED event needs the server to
// send a MonsterDamage packet, i.e. whether the damage number has no
// client-side rendering of its own.
//
//   - CHARACTER_ATTACK: no. Observers already render it from the attack
//     broadcast (CharacterAttack*Writer,
//     socket/handler/character_attack_common.go).
//   - DAMAGE_OVER_TIME: no. The client runs a poison tick on its own timer and
//     renders the number from the POISON magnitude carried in the monster
//     temporary-stat packet (handleStatusEffectApplied), which atlas-monsters
//     resolves to the real per-tick damage. Echoing here double-renders it.
//   - HEAL: no. Emitted purely so the HP bar refreshes; it carries no damage.
//   - MONSTER_ATTACK: yes. Nothing else renders it.
//
// The HP-bar packet, by contrast, is server-driven for every source.
func shouldEchoDamagePacket(damageSource string) bool {
	return damageSource == monster2.DamageSourceMonsterAttack
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
		routine.Go(l, ctx, func(_ context.Context) {
			if e.Body.Boss {
				err = _map.NewProcessor(l, ctx).ForSessionsInMap(f, announcer)
			} else {
				idProvider := model2.FixedProvider([]uint32{e.Body.ActorId})

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
		})
		if shouldEchoDamagePacket(e.Body.DamageSource) {
			routine.Go(l, ctx, func(_ context.Context) {
				err = _map.NewProcessor(l, ctx).ForSessionsInMap(f, func(s session.Model) error {
					return session.Announce(l)(ctx)(wp)(monsterpkt.MonsterDamageWriter)(monsterpkt.NewMonsterDamage(m.UniqueId(), monsterpkt.MonsterDamageTypeUnk3, e.Body.Damage, m.Hp(), m.MaxHp()).Encode)(s)
				})
			})
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
		monster.GetLiveMirror().Remove(tenant.MustFromContext(ctx), e.UniqueId)
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

		monster.GetLiveMirror().UpdateAggro(tenant.MustFromContext(ctx), e.UniqueId, e.Body.ControllerHasAggro)
		monster.GetLiveMirror().UpdateControl(tenant.MustFromContext(ctx), e.UniqueId, e.Body.ActorId)

		// Prefer the authoritative model: the event envelope carries no status
		// effects, and the Spawn/Control bodies both encode a temporary-stat
		// block. That matters beyond cosmetics here — the map-enter fast path
		// (map consumer spawnMonsterForSession) may already have sent a Spawn
		// carrying real temporary stats, and CMob::SetTemporaryStat resets the
		// block before decoding, so an envelope-derived Spawn would wipe them.
		// Falling back to the envelope on a fetch failure still beats dropping
		// the grant, which is the bug this whole path exists to fix.
		m, err := monsterGetByIdFn(l, ctx, e.UniqueId)
		if err != nil {
			degrade.Observe(l, "channel.monster.control_grant_fetch", e.UniqueId, err)
			l.WithError(err).Warnf("Unable to retrieve monster [%d] for control grant; falling back to the event envelope.", e.UniqueId)
			f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
			m = monster.NewModelBuilder(e.UniqueId, f, e.MonsterId).
				SetControlCharacterId(e.Body.ActorId).
				SetX(e.Body.X).SetY(e.Body.Y).
				SetStance(e.Body.Stance).
				SetFH(e.Body.FH).
				SetTeam(e.Body.Team).
				MustBuild()
		}

		if err := controlGrantFn(l, ctx, sc, wp, m, e.Body.ControllerHasAggro, e.Body.ActorId); err != nil {
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

		// No controller => no aggro (design §5.2).
		monster.GetLiveMirror().UpdateAggro(tenant.MustFromContext(ctx), e.UniqueId, false)
		monster.GetLiveMirror().UpdateControl(tenant.MustFromContext(ctx), e.UniqueId, 0)

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

		monster.GetLiveMirror().UpdateAggro(tenant.MustFromContext(ctx), e.UniqueId, e.Body.ControllerHasAggro)

		m, err := monsterGetByIdFn(l, ctx, e.UniqueId)
		if err != nil {
			l.WithError(err).Errorf("Unable to retrieve monster [%d] for aggro change.", e.UniqueId)
			return
		}
		sf := func(s session.Model) error {
			return announceFn(l, ctx, wp, monsterpkt.MonsterControlWriter, writer.StartControlMonsterBody(m, e.Body.ControllerHasAggro), s)
		}
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

// monsterGetByIdFn is the REST-fetch seam for handleStatusEventCreated so
// tests can seed the live mirror without an HTTP fake (same pattern as the
// broadcaster spy vars below).
var monsterGetByIdFn = func(l logrus.FieldLogger, ctx context.Context, uniqueId uint32) (monster.Model, error) {
	return monster.NewProcessor(l, ctx).GetById(uniqueId)
}

// controlGrantFn delivers controller ownership of one mob to one character as
// Spawn-then-Control, in that order, on a single session.
//
// The ordering is load-bearing, not defensive. On v79 (CMobPool::SetLocalMob
// 0x645ce1) and v83 (0x678308) a MonsterControl for an unknown mob is NOT
// dropped: GetMob misses, and the client materializes the mob from the Control
// body via CreateMob -> CMob::Init. A 0/1 stance then routes into
// CMob::OnResolveMoveAction and null-derefs, and a control-first birth on a
// slope lands the mob below the surface. Sending Spawn first makes that
// impossible.
//
// The leading Spawn is safe when the client already has the mob:
// CMobPool::OnMobEnterField (v83 0x67945a, v79 0x646e33) takes its GetMob-hit
// branch, which sets m_bInViewSplit and calls CMob::SetTemporaryStat and
// nothing else — no CMob::Init, no reposition, no SetActive, no
// CMovePath::DiscardByInterrupt. Position, stance and in-flight movement are
// untouched. SetTemporaryStat re-bases the mob's temp-stat block, which
// SetLocalMob already does on every controller change anyway.
//
// A duplicate grant is possible: the map-enter fast path
// (map consumer spawnMonsterForSession) and this handler can both fire for the
// same mob and session when the controller assignment lands before the
// channel's monster-list read. Each path is internally ordered, so the
// Spawn-before-Control invariant still holds. The repeat Control is near-inert
// on the client — SetLocalMob's GetMob hit branch runs SetTemporaryStat, then
// CMob::SetActive(1), which self-guards on the already-active flag (v83
// 0x6637ec). The one live effect is CMob::ChaseTarget, re-issued when the
// control type exceeds 1, i.e. only for an aggro grant; it re-targets an
// already-chasing mob rather than changing its state.
//
// Package-level seam so tests can assert grant delivery without standing up a
// session; the ordering itself lives in spawnThenControlOperator.
var controlGrantFn = func(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, m monster.Model, aggro bool, characterId uint32) error {
	return session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(characterId, spawnThenControlOperator(l, ctx, wp, m, aggro))
}

// announceFn is the single-packet announce seam, held as a var so
// spawnThenControlOperator's packet ordering is assertable in a unit test.
var announceFn = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, writerName string, body packet.Encode, s session.Model) error {
	return session.Announce(l)(ctx)(wp)(writerName)(body)(s)
}

// spawnThenControlOperator emits Spawn followed by Control on one session. The
// order is the whole point — see controlGrantFn.
func spawnThenControlOperator(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, m monster.Model, aggro bool) model2.Operator[session.Model] {
	return func(s session.Model) error {
		if err := announceFn(l, ctx, wp, monsterpkt.MonsterSpawnWriter, writer.SpawnMonsterBody(m, false), s); err != nil {
			return err
		}
		return announceFn(l, ctx, wp, monsterpkt.MonsterControlWriter, writer.StartControlMonsterBody(m, aggro), s)
	}
}

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

		// Mirror MP before any session gating or Reason dispatch so the live
		// mirror tracks every MP mutation — including Reasons this handler
		// doesn't otherwise act on and events whose character has no local
		// session (design §5.2).
		monster.GetLiveMirror().UpdateMp(tenant.MustFromContext(ctx), e.UniqueId, e.Body.MonsterMpAfter)

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

// bridleFailReason maps an internal catch-failure cause onto the client's wire
// reason byte and reports whether to send the packet at all. The wire value is
// resolved HERE, in the rendering service — the domain services emit semantic
// causes only (DOM-25).
func bridleFailReason(cause string) (byte, bool) {
	switch cause {
	case consumable2.CatchCauseUseDelay:
		return 1, true
	case monster2.CatchCauseUnresolved:
		return 0, false
	default:
		return 0, true
	}
}

// handleStatusEventCaught renders a successful capture to everyone in the map,
// then unlocks the acting character alone.
//
// ONE effect packet, not two. The client has two independent renderers for a
// capture and neither reads bridleMsgType off the wire, so the server chooses:
//
//   - CATCH_MONSTER_WITH_ITEM (CMob::OnEffectByItem @v83 0x66d997) plays the
//     item-keyed animation, Effect/ItemEff.img/<itemId> via
//     CAnimationDisplayer::Effect_ByItem @0x438b36, at the mob's y-2, and plays
//     the item's sound. This is the one an item-initiated catch wants — it is
//     the render observed in the reference client footage.
//   - CATCH_MONSTER (CMob::OnCatchEffect @v83 0x66d6b9) plays a generic
//     capture image out of Effect/BasicEff.img (Effect_Catch @0x438eb6:
//     StringPool 3687 when result != 0, 3688 when 0) at the mob's y-15.
//
// Sending both, as this handler originally did, stacked two animations on one
// capture. The generic one is dropped here, and it is very likely not a capture
// render at all: ShowCatchEffect has a second, purely client-side caller in
// CMob::OnHit (v83 0x668b83, call at 0x668e22), reached only when the hitting
// skill is 1121001/1221001/1321001 — Hero/Paladin/DarkKnight Monster Magnet —
// with its argument being (grab result == 3). So Effect_Catch is the
// Monster-Magnet grab succeeded/failed image, which the client plays for itself
// on the magnet path; CATCH_MONSTER is the server-driven entry to that same
// renderer, not a bridle-capture effect.
//
// The CatchMonster codec and its template routes are deliberately retained: it
// is a real protocol element with no sender today, and re-adding it is one
// announce.
//
// This MUST reach the client before the sibling DESTROYED event: the client
// resolves the mob via CMobPool::OnMobPacket -> GetMob and silently drops the
// packet once the mob is gone. Both events are keyed by MapId on the same topic,
// so the ordering is a partition guarantee.
func handleStatusEventCaught(sc server.Model, wp writer.Producer) message.Handler[monster2.StatusEvent[monster2.StatusEventCaughtBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monster2.StatusEvent[monster2.StatusEventCaughtBody]) {
		if e.Type != monster2.EventStatusCaught {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		f := sc.Field(e.MapId, e.Instance)
		if err := _map.NewProcessor(l, ctx).ForSessionsInMap(f,
			session.Announce(l)(ctx)(wp)(monsterpkt.CatchMonsterWithItemWriter)(writer.CatchMonsterWithItemBody(e.UniqueId, int32(e.Body.ItemId), 1)),
		); err != nil {
			l.WithError(err).Errorf("Unable to announce the capture of monster [%d] in map [%d].", e.UniqueId, e.MapId)
		}

		// Emitted from its own statement so a failed effect broadcast can never
		// leave the client wedged.
		if err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.Body.CharacterId, session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)); err != nil {
			l.WithError(err).Errorf("Unable to unlock character [%d] after a successful catch.", e.Body.CharacterId)
		}
	}
}

// handleStatusEventCatchFailed renders a failed capture to the acting character
// only. The fail packet is optional — gms_v48 has no OnBridleMobCatchFail
// handler at all (its writer is simply not routed, and the writer registry
// reports it unconfigured) and UNRESOLVED deliberately renders nothing — but the
// unlock is not, so it is emitted from its own statement.
func handleStatusEventCatchFailed(sc server.Model, wp writer.Producer) message.Handler[monster2.StatusEvent[monster2.StatusEventCatchFailedBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e monster2.StatusEvent[monster2.StatusEventCatchFailedBody]) {
		if e.Type != monster2.EventStatusCatchFailed {
			return
		}
		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}
		AnnounceCatchFailure(l, ctx, sc, wp, e.Body.CharacterId, e.Body.ItemId, e.Body.Cause)
	}
}

// AnnounceCatchFailure is shared by the monster-side and consumable-side
// failure paths (kafka/consumer/consumable) so both render identically and
// both always unlock. Exported to avoid an import cycle: the consumable
// consumer cannot depend on this package's unexported helpers.
func AnnounceCatchFailure(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, characterId uint32, itemId uint32, cause string) {
	sp := session.NewProcessor(l, ctx)
	if reason, send := bridleFailReason(cause); send {
		if err := sp.IfPresentByCharacterId(sc.Channel())(characterId, session.Announce(l)(ctx)(wp)(charpkt.BridleMobCatchFailWriter)(writer.BridleMobCatchFailBody(reason, int32(itemId), 0))); err != nil {
			l.WithError(err).Debugf("Unable to write [%s] for character [%d]; continuing to the unlock.", charpkt.BridleMobCatchFailWriter, characterId)
		}
	}
	if err := sp.IfPresentByCharacterId(sc.Channel())(characterId, session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)); err != nil {
		l.WithError(err).Errorf("Unable to unlock character [%d] after a failed catch.", characterId)
	}
}
