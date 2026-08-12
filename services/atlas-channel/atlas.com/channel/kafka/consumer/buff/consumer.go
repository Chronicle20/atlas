package buff

import (
	"atlas-channel/battleship"
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/buff/stat"
	npc2 "atlas-channel/data/npc"
	dataskill "atlas-channel/data/skill"
	"atlas-channel/data/skill/effect/statup"
	consumer2 "atlas-channel/kafka/consumer"
	buff2 "atlas-channel/kafka/message/buff"
	"atlas-channel/listener"
	_map "atlas-channel/map"
	controllernpc "atlas-channel/npc/controller"
	"atlas-channel/server"
	"atlas-channel/session"
	socketHandler "atlas-channel/socket/handler"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_buff_status_event")(buff2.EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(buff2.EnvEventStatusTopic)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventApplied(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventExpired(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventStatUpdated(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventGmHideApplied(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventGmHideExpired(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventBerserk(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

// announceBuffGive sends the buff stat set to the owner (GIVE_BUFF) and to
// all other sessions in the owner's map (GIVE_FOREIGN_BUFF). Shared by the
// APPLIED and STAT_UPDATED handlers — the packet layer derives the client
// duration from expiresAt, so callers passing a buff's original timestamps
// broadcast the remaining duration.
func announceBuffGive(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, characterId uint32, sourceId int32, level byte, duration int32, statChanges []buff2.StatChange, createdAt time.Time, expiresAt time.Time, noExpiry bool) {
	_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(characterId, func(s session.Model) error {
		bs := make([]buff.Model, 0)
		changes := make([]stat.Model, 0)
		for _, cm := range statChanges {
			changes = append(changes, stat.NewStat(cm.Type, cm.Amount))
		}
		bs = append(bs, buff.NewBuff(sourceId, level, duration, changes, createdAt, expiresAt, noExpiry))

		err := session.Announce(l)(ctx)(wp)(charpkt.CharacterBuffGiveWriter)(writer.CharacterBuffGiveBody(bs))(s)
		if err != nil {
			l.WithError(err).Errorf("Unable to write new character [%d] buffs.", characterId)
		}

		_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), func(os session.Model) error {
			err = session.Announce(l)(ctx)(wp)(charpkt.CharacterBuffGiveForeignWriter)(writer.CharacterBuffGiveForeignBody(characterId, bs))(os)
			if err != nil {
				l.WithError(err).Errorf("Unable to write new character [%d] buffs.", characterId)
				return err
			}
			return nil
		})
		return nil
	})
}

func handleStatusEventApplied(sc server.Model, wp writer.Producer) message.Handler[buff2.StatusEvent[buff2.AppliedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.AppliedStatusEventBody]) {
		if e.Type != buff2.EventStatusTypeBuffApplied {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !sc.IsWorld(t, e.WorldId) {
			return
		}

		// Battleship ride begins: record the pod-local riding truth the
		// damage/attack hot paths read via battleship.Processor.IsRiding
		// (mirror; FR-3.1/FR-6.2). Gated on session presence in this
		// channel's local registry — like the announce below, this is how
		// a world-broadcast buff event is scoped to the one channel pod
		// that actually owns the socket (RideMirror is per-channel-process;
		// see battleship/mirror.go). Routed through the same
		// newBattleshipProcessor seam as the EXPIRED hook's Clear below, so
		// both ends of the ride lifecycle stay behind the Processor
		// boundary and remain independently testable.
		if isBattleshipRide(e.Body.SourceId, e.Body.Changes) {
			_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
				newBattleshipProcessor(l, ctx).StartRide(e.CharacterId, battleship.RideState{
					SkillLevel: e.Body.Level,
					StateTTL:   battleshipStateTTLFunc(l, ctx, e.Body.Level),
				})
				return nil
			})
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			t := tenant.MustFromContext(ctx)
			// Track the active beacon from its own APPLIED event.
			if bc, ok := beaconChange(e.Body.Changes); ok {
				buff.GetBeaconMirror().Set(t, e.CharacterId, buff.NewBeaconEntry(e.Body.SourceId, e.Body.Level, bc.Amount))
			}

			if ec, ok := energyChargeChange(e.Body.Changes); ok {
				energyChargeReact(l, ctx, sc, wp, e.CharacterId, e.Body.SourceId, e.Body.Level, ec)
			}

			bs := make([]buff.Model, 0)
			changes := make([]stat.Model, 0)
			for _, cm := range e.Body.Changes {
				changes = append(changes, stat.NewStat(cm.Type, cm.Amount))
			}
			bs = append(bs, buff.NewBuff(e.Body.SourceId, e.Body.Level, e.Body.Duration, changes, e.Body.CreatedAt, e.Body.ExpiresAt, e.Body.NoExpiry))

			// F2: while locked, every LOCAL give must re-carry the populated
			// beacon block (pre-95 clients overwrite the stored beacon from
			// every local give trailer). Skip when this event is itself the
			// beacon — bs already carries it.
			localBs := bs
			if _, isBeacon := beaconChange(e.Body.Changes); !isBeacon {
				if entry, ok := buff.GetBeaconMirror().Get(t, e.CharacterId); ok {
					localBs = mergeBeacon(bs, entry)
				}
			}

			err := session.Announce(l)(ctx)(wp)(charpkt.CharacterBuffGiveWriter)(writer.CharacterBuffGiveBody(localBs))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write new character [%d] buffs.", e.CharacterId)
			}

			// Beacon-only events are never announced to other players: the
			// stat is caster-only and the remote GuidedBullet read path is
			// unverified (FR-4.5). The foreign body uses the UNMERGED bs.
			if !isBeaconOnly(e.Body.Changes) {
				_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), func(os session.Model) error {
					err = session.Announce(l)(ctx)(wp)(charpkt.CharacterBuffGiveForeignWriter)(writer.CharacterBuffGiveForeignBody(e.CharacterId, bs))(os)
					if err != nil {
						l.WithError(err).Errorf("Unable to write new character [%d] buffs.", e.CharacterId)
						return err
					}
					return nil
				})
			}
			return nil
		})
	}
}

func handleStatusEventStatUpdated(sc server.Model, wp writer.Producer) message.Handler[buff2.StatusEvent[buff2.StatUpdatedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.StatUpdatedStatusEventBody]) {
		if e.Type != buff2.EventStatusTypeStatUpdated {
			return
		}

		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}

		if ec, ok := energyChargeChange(e.Body.Changes); ok {
			energyChargeReact(l, ctx, sc, wp, e.CharacterId, e.Body.SourceId, e.Body.Level, ec)
		}

		// StatUpdatedStatusEventBody carries no NoExpiry field (task-167 FR-2
		// scoped the flag to APPLY/APPLIED/EXPIRED only) — this transient
		// re-broadcast buff is display-only (see CharacterBuffGiveBody, which
		// reads ExpiresAt() directly and never calls Expired()), so false is
		// a safe default.
		announceBuffGive(l, ctx, sc, wp, e.CharacterId, e.Body.SourceId, e.Body.Level, e.Body.Duration, e.Body.Changes, e.Body.CreatedAt, e.Body.ExpiresAt, false)
	}
}

func handleStatusEventExpired(sc server.Model, wp writer.Producer) message.Handler[buff2.StatusEvent[buff2.ExpiredStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.ExpiredStatusEventBody]) {
		if e.Type != buff2.EventStatusTypeBuffExpired {
			return
		}

		t := tenant.MustFromContext(ctx)
		if !sc.IsWorld(t, e.WorldId) {
			return
		}

		session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			// Battleship ride ends (manual dismount toggle, server cancel on
			// break, or natural expiry): clear mirror + ship HP state
			// (FR-5.1). NO cooldown here — breakShip in the battleship
			// package is the only cooldown trigger (FR-4.3); this hook only
			// clears state.
			if isBattleshipRide(e.Body.SourceId, e.Body.Changes) {
				newBattleshipProcessor(l, ctx).Clear(e.CharacterId)
			}

			if _, ok := beaconChange(e.Body.Changes); ok {
				buff.GetBeaconMirror().Clear(t, e.CharacterId)
			}

			if _, ok := energyChargeChange(e.Body.Changes); ok {
				buff.GetEnergyMirror().Clear(t, e.CharacterId)
			}

			ebs := make([]buff.Model, 0)
			changes := make([]stat.Model, 0)
			for _, cm := range e.Body.Changes {
				changes = append(changes, stat.NewStat(cm.Type, cm.Amount))
			}
			ebs = append(ebs, buff.NewBuff(e.Body.SourceId, e.Body.Level, e.Body.Duration, changes, e.Body.CreatedAt, e.Body.ExpiresAt, e.Body.NoExpiry))

			err := session.Announce(l)(ctx)(wp)(charpkt.CharacterBuffCancelWriter)(writer.CharacterBuffCancelBody(ebs))(s)
			if err != nil {
				l.WithError(err).Errorf("Unable to write character [%d] cancelled buffs.", e.CharacterId)
			}

			// Beacon-only events are never announced to other players: the
			// stat is caster-only and the remote GuidedBullet read path is
			// unverified (FR-4.5).
			if !isBeaconOnly(e.Body.Changes) {
				_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), func(os session.Model) error {
					err = session.Announce(l)(ctx)(wp)(charpkt.CharacterBuffCancelForeignWriter)(writer.CharacterBuffCancelForeignBody(e.CharacterId, ebs))(os)
					if err != nil {
						l.WithError(err).Errorf("Unable to write new character [%d] buffs.", e.CharacterId)
						return err
					}
					return nil
				})
			}
			return nil
		})
	}
}

// isSuperGmHideSource reports whether sourceId -- a buff's version-specific
// WIRE skill id (5101004 at v0.48, 9101004 at v0.62+) -- resolves to the
// SuperGmHide identity under t's version set (task-187). A raw compare
// against the canonical wire constant would silently never match a v0.48
// hide buff.
func isSuperGmHideSource(t tenant.Model, sourceId int32) bool {
	set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion())
	id, ok := set.Skill.Resolve(skill2.Id(sourceId))
	return ok && id == skill2.SuperGmHide
}

// handleStatusEventGmHideApplied relinquishes the hiding GM's NPCs
// (task-176, FR-6.1): revoke their client-side grants, then reassign to a
// visible session. Fires ONLY for SuperGmHide; Dark Sight and all other
// buffs are untouched.
func handleStatusEventGmHideApplied(sc server.Model, wp writer.Producer) message.Handler[buff2.StatusEvent[buff2.AppliedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.AppliedStatusEventBody]) {
		if e.Type != buff2.EventStatusTypeBuffApplied {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !isSuperGmHideSource(t, e.Body.SourceId) {
			return
		}
		if !sc.IsWorld(t, e.WorldId) {
			return
		}
		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			f := s.Field()
			cp := controllernpc.NewProcessor(l, ctx)
			released, err := cp.ReleaseFor(f, s.CharacterId())
			if err != nil {
				l.WithError(err).Warnf("GM-hide: unable to release NPC controller entries for [%d] in field [%s].", s.CharacterId(), f.Id())
				return nil
			}
			if len(released) == 0 {
				l.Debugf("GM-hide: character [%d] controlled no NPCs in field [%s].", s.CharacterId(), f.Id())
				return nil
			}
			for _, npcId := range released {
				if rerr := controllernpc.AnnounceRevoke(l, ctx, wp)(s, npcId); rerr != nil {
					l.WithError(rerr).Warnf("GM-hide: unable to revoke NPC [%d] control from [%d].", npcId, s.CharacterId())
				}
			}
			assignments, aerr := cp.ElectFor(f, released, s.CharacterId())
			if aerr != nil {
				l.WithError(aerr).Warnf("GM-hide: unable to re-elect NPC controllers in field [%s].", f.Id())
				return nil
			}
			for npcId, winner := range assignments {
				if gerr := controllernpc.AnnounceGrant(l, ctx, wp)(f, winner, npcId); gerr != nil {
					l.WithError(gerr).Warnf("GM-hide: unable to announce NPC [%d] grant to [%d].", npcId, winner)
				}
			}
			l.Debugf("GM-hide: character [%d] relinquished [%d] NPCs in field [%s]; reassigned [%d].", s.CharacterId(), len(released), f.Id(), len(assignments))
			return nil
		})
	}
}

// handleStatusEventGmHideExpired restores the revealed GM's candidacy
// (FR-6.3): elect controllers for currently-uncontrolled NPCs with the GM
// back in the pool. No forced transfer — live controllers keep their NPCs.
// (atlas-buffs prunes its registry BEFORE emitting EXPIRED, so the
// winner-check cannot see a stale hide buff.)
func handleStatusEventGmHideExpired(sc server.Model, wp writer.Producer) message.Handler[buff2.StatusEvent[buff2.ExpiredStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.ExpiredStatusEventBody]) {
		if e.Type != buff2.EventStatusTypeBuffExpired {
			return
		}
		t := tenant.MustFromContext(ctx)
		if !isSuperGmHideSource(t, e.Body.SourceId) {
			return
		}
		if !sc.IsWorld(t, e.WorldId) {
			return
		}
		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			f := s.Field()
			// Use InMapModelProvider + a sequential range rather than
			// ForEachInMap: the latter runs its callback via
			// model.ForEachSlice(..., model.ParallelExecute()) — one
			// goroutine per NPC — so accumulating into npcIds from inside
			// that callback would race on the shared slice header
			// (task-176 review; same bug class as the Task 9 hiddenCache
			// race fixed in e6c75ed42). InMapModelProvider(...)() fetches
			// the slice synchronously; the append loop below runs on this
			// goroutine only.
			npcs, err := npc2.NewProcessor(l, ctx).InMapModelProvider(f.MapId())()
			if err != nil {
				l.WithError(err).Warnf("GM-reveal: unable to enumerate NPCs in map [%d].", f.MapId())
				return nil
			}
			npcIds := make([]uint32, 0, len(npcs))
			for _, n := range npcs {
				npcIds = append(npcIds, n.Id())
			}
			cp := controllernpc.NewProcessor(l, ctx)
			unc, err := cp.UncontrolledIn(f, npcIds)
			if err != nil {
				l.WithError(err).Warnf("GM-reveal: unable to compute uncontrolled NPCs in field [%s].", f.Id())
				return nil
			}
			if len(unc) == 0 {
				return nil
			}
			assignments, aerr := cp.ElectFor(f, unc)
			if aerr != nil {
				l.WithError(aerr).Warnf("GM-reveal: unable to elect NPC controllers in field [%s].", f.Id())
				return nil
			}
			for npcId, winner := range assignments {
				if gerr := controllernpc.AnnounceGrant(l, ctx, wp)(f, winner, npcId); gerr != nil {
					l.WithError(gerr).Warnf("GM-reveal: unable to announce NPC [%d] grant to [%d].", npcId, winner)
				}
			}
			l.Debugf("GM-reveal: elected controllers for [%d] of [%d] uncontrolled NPCs in field [%s].", len(assignments), len(unc), f.Id())
			return nil
		})
	}
}

// handleStatusEventBerserk translates one berserk broadcast tick into the own
// + foreign EffectSkillUse packets (task-154). Stateless by design (D4):
// atlas-buffs owns the schedule; the periodic re-broadcast covers late-joining
// observers, so there is no map-enter hook. No session means the character
// transferred or logged out between emit and consume — the next tick
// self-corrects.
func handleStatusEventBerserk(sc server.Model, wp writer.Producer) message.Handler[buff2.StatusEvent[buff2.BerserkStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.BerserkStatusEventBody]) {
		if e.Type != buff2.EventStatusTypeBerserk {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.Body.ChannelId) {
			return
		}

		_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			if err := socketHandler.AnnounceBerserkEffect(l)(ctx)(wp)(e.Body.SkillId, e.Body.CharacterLevel, e.Body.SkillLevel, e.Body.Active)(s); err != nil {
				l.WithError(err).Errorf("Unable to write berserk effect for character [%d].", e.CharacterId)
			}

			_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), socketHandler.AnnounceForeignBerserkEffect(l)(ctx)(wp)(e.CharacterId, e.Body.SkillId, e.Body.CharacterLevel, e.Body.SkillLevel, e.Body.Active))
			return nil
		})
	}
}

// beaconChange returns the first HOMING_BEACON stat change carried by an
// event, if any.
func beaconChange(changes []buff2.StatChange) (buff2.StatChange, bool) {
	for _, c := range changes {
		if c.Type == string(charconst.TemporaryStatTypeHomingBeacon) {
			return c, true
		}
	}
	return buff2.StatChange{}, false
}

// isBeaconOnly reports whether an event's changes carry nothing but the
// beacon stat. Such events are never announced to other players: the stat is
// caster-only and the foreign GuidedBullet read path is unverified (FR-4.5).
func isBeaconOnly(changes []buff2.StatChange) bool {
	if len(changes) == 0 {
		return false
	}
	for _, c := range changes {
		if c.Type != string(charconst.TemporaryStatTypeHomingBeacon) {
			return false
		}
	}
	return true
}

// mergeBeacon appends the character's active beacon as a synthetic no-expiry
// buff so an unrelated local give re-carries the populated GuidedBullet block
// (pre-95 clients overwrite the stored beacon from every local give trailer —
// design.md §3 F2). Idempotent client-side: SetGuided on the same mob is a
// re-apply.
func mergeBeacon(bs []buff.Model, e buff.BeaconEntry) []buff.Model {
	return append(bs, buff.NewBuff(e.SourceId(), e.Level(), 0,
		[]stat.Model{stat.NewStat(string(charconst.TemporaryStatTypeHomingBeacon), e.MobId())},
		time.Now(), time.Time{}, true))
}

const (
	// Sourced from libs/atlas-constants rather than re-declared, so the
	// consumer agrees with the socket handler's accumulation ceiling without
	// depending on the socket handler package. The charged value is a SENTINEL,
	// not a bar reading.
	energyChargeCapValue = charconst.EnergyChargeCap
	energyChargedValue   = charconst.EnergyChargedValue
)

// energyChargeChange returns the event's ENERGY_CHARGE stat change, if any.
func energyChargeChange(changes []buff2.StatChange) (buff2.StatChange, bool) {
	for _, c := range changes {
		if c.Type == string(charconst.TemporaryStatTypeEnergyCharge) {
			return c, true
		}
	}
	return buff2.StatChange{}, false
}

// energyChargeShouldPromote reports whether a bar reading is the one that
// tops the accumulation cap and therefore triggers the charged state.
func energyChargeShouldPromote(amount int32) bool {
	return amount == energyChargeCapValue
}

// energyChargeReact is the whole Energy Charge reaction to one buff-status
// event carrying an ENERGY_CHARGE change: refresh the pod-local mirror the
// cast gate reads, announce the skill-use effect to the owner and the map,
// and — when the bar just topped out — promote to the charged state.
//
// Announcing the effect HERE rather than at the attack site is what keeps it
// honest: atlas-buffs emits a status event only when the value actually
// changed, so a hit against a full bar produces no packet.
func energyChargeReact(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, characterId uint32, sourceId int32, level byte, c buff2.StatChange) {
	t := tenant.MustFromContext(ctx)
	buff.GetEnergyMirror().Set(t, characterId, c.Amount)

	_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(characterId, func(s session.Model) error {
		cp := character.NewProcessor(l, ctx)
		ch, cerr := cp.GetById()(characterId)
		if cerr != nil {
			l.WithError(cerr).Errorf("Energy Charge: unable to read character [%d] for the skill-use effect.", characterId)
		} else {
			if aerr := socketHandler.AnnounceSkillUse(l)(ctx)(wp)(uint32(sourceId), ch.Level(), level)(s); aerr != nil {
				l.WithError(aerr).Errorf("Energy Charge: skill-use effect write failed for character [%d].", characterId)
			}
			_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), characterId,
				socketHandler.AnnounceForeignSkillUse(l)(ctx)(wp)(characterId, uint32(sourceId), ch.Level(), level))
		}

		if !energyChargeShouldPromote(c.Amount) {
			return nil
		}

		// The charged window's length is the Energy Charge effect's `time` at
		// the character's skill level (31s at L1, 40s at L20 for 5110001; the
		// Cygnus table differs, which is why the level travels with the event
		// rather than being assumed). Duration() ALREADY returns milliseconds
		// and ApplyCommandBody.Duration is milliseconds — no scaling here.
		// tools/buff-duration-guard.sh fails CI on a seconds-valued emitter.
		se, eerr := dataskill.NewProcessor(l, ctx).GetEffect(uint32(sourceId), level)
		if eerr != nil {
			l.WithError(eerr).Errorf("Energy Charge: effect lookup failed for character [%d] skill [%d] level [%d]; the bar stays full but uncharged.", characterId, sourceId, level)
			return nil
		}

		// The charged APPLY REPLACES the accumulating buff in place: both
		// phases share srcKey(sourceId) in atlas-buffs, so there is never a
		// moment with two Energy Charge buffs.
		if perr := buff.NewProcessor(l, ctx).Apply(s.Field(), characterId, sourceId, level, se.Duration(),
			[]statup.Model{statup.NewModel(string(charconst.TemporaryStatTypeEnergyCharge), energyChargedValue)})(characterId); perr != nil {
			l.WithError(perr).Errorf("Energy Charge: charged APPLY emit failed for character [%d].", characterId)
		}
		return nil
	})
}

// isBattleshipRide reports whether a buff status event is the battleship
// mount buff (MONSTER_RIDING sourced from 5221006).
func isBattleshipRide(sourceId int32, changes []buff2.StatChange) bool {
	if sourceId != int32(skill2.CorsairBattleshipId) {
		return false
	}
	for _, c := range changes {
		if c.Type == string(charconst.TemporaryStatTypeMonsterRiding) {
			return true
		}
	}
	return false
}

// newBattleshipProcessor is a seam over battleship.NewProcessor (same
// pattern as battleshipStateTTLFunc below) so tests can substitute a spy
// implementing battleship.Processor and assert EXACTLY which methods the
// ride-end hook invokes — in particular, that it calls Clear and never
// Drain, the only path that can reach breakShip's cooldown emit.
var newBattleshipProcessor = battleship.NewProcessor

// battleshipStateTTLFunc derives the ship-state TTL from the effect's buff
// duration (FR-5.2). Returns 0 on failure — the battleship package falls
// back to its own default. Seam for tests.
var battleshipStateTTLFunc = func(l logrus.FieldLogger, ctx context.Context, level byte) time.Duration {
	e, err := dataskill.NewProcessor(l, ctx).GetEffect(uint32(skill2.CorsairBattleshipId), level)
	if err != nil || e.Duration() <= 0 {
		if err != nil {
			l.WithError(err).Warnf("Unable to derive battleship state TTL from effect level [%d]; using fallback.", level)
		}
		return 0
	}
	return time.Duration(e.Duration()) * time.Millisecond
}
