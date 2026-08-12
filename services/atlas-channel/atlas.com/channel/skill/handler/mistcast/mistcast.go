// Package mistcast holds the cast-time logic every player-cast mist skill
// shares: validate the effect, load the caster's position, emit CREATE to
// atlas-maps. The five mist skills (Poison Mist, Flame Gear, Poison Bomb,
// Smokescreen, Recovery Aura) differ only in the Params they build, so the
// validation rules -- each of which encodes an expensively-learned client
// behaviour -- live here once rather than in five copies.
package mistcast

import (
	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	"atlas-channel/mist"
	"context"

	mistmsg "atlas-channel/kafka/message/mist"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// PlayerMistTickIntervalMs is the RE-APPLY cadence of a player-cast mist --
// how often atlas-maps re-issues its per-tick effect to everything still
// inside the cloud (mist.Mist.ShouldTick). It is NOT the DoT damage cadence
// and does NOT flow into atlas-monsters' DoT gate.
//
// atlas-maps sends a SEPARATE, strictly smaller DoT tick interval on every
// APPLY_STATUS command (monsterDotTickIntervalMs,
// services/atlas-maps/atlas.com/maps/tasks/mist_tick.go, currently 1000ms).
// atlas-monsters' StatusEffect.ShouldTick gates actual damage on
// `since(lastTick) >= tickInterval`
// (services/atlas-monsters/atlas.com/monsters/monster/status.go:129-134).
//
// It must be an interval, not a WZ value: no `dotInterval` node exists in any
// provisioned Skill.wz for these skills (task-200 design §2.1; task-218
// design §6.3 for Recovery Aura, whose dot/dotInterval/dotTime are all 0).
//
// This value MUST exceed the DoT tick interval sent by atlas-maps.
// atlas-monsters' ModelBuilder.AddStatusEffect REPLACES a same-type POISON on
// every re-apply with a fresh StatusEffect whose lastTick = now
// (services/atlas-monsters/atlas.com/monsters/monster/builder.go:141-163,
// services/atlas-monsters/atlas.com/monsters/monster/status.go:35-49). So the
// eligible damage window per re-apply cycle is `PlayerMistTickIntervalMs (P)
// - monsterDotTickIntervalMs (T)` wide, NOT P. A prior fix attempt set
// atlas-maps' emitted TickInterval to this same constant (P == T by
// construction), which makes that window exactly 0 regardless of P's value --
// the mist would never deal damage no matter how this constant was tuned.
// With P = 3000ms and T = 1000ms the window is a genuine 2000ms per cycle.
const PlayerMistTickIntervalMs int64 = 3000

// MaxPlayerMistDurationMs rejects (never truncates) an implausible mist
// lifetime. The largest legitimate `time` across all five mist skills and all
// provisioned versions is Smokescreen's 60s at level 30, so this 5-minute
// ceiling is 5x the largest real value and can only fire on corrupt or
// mis-scaled data.
//
// This is deliberately NOT atlas-monsters' 60s MistDurationCapMs. A clamp
// would desynchronise the client, which computes its own
// tEnd = tStart + 1000*SKILLLEVELDATA::tTime from its own WZ (v83 @0x43200f,
// v95 @0x437c95) and would keep rendering a mist the server stopped ticking.
const MaxPlayerMistDurationMs int32 = 300_000

// Params is everything a mist cast differs by. Everything else -- the
// rectangle, the lifetime, the origin, the source ids -- is derived from the
// effect and the cast itself.
type Params struct {
	// SkillName names the skill in log lines. Rejections must be traceable
	// to a skill without decoding an id.
	SkillName string
	// TargetKind / EffectKind are the mist contract's descriptors.
	TargetKind string
	EffectKind string
	// Disease is the status name a DAMAGE_OVER_TIME mist applies; empty for
	// every other kind.
	Disease string
	// TickMs is the mist's re-apply cadence. 0 means "never ticks", which is
	// how a PROTECTION mist is expressed -- it is evaluated on the channel's
	// damage path, not by an atlas-maps tick.
	TickMs int64
	// RecoveryMp is the per-tick MP a RECOVERY mist restores; 0 otherwise.
	RecoveryMp int32
	// PartyMemberIds scopes a RECOVERY mist; nil otherwise.
	PartyMemberIds []uint32
}

// Seams are the two external effects a cast performs. Each handler keeps its
// own package-level vars and passes them here so its tests can record
// instead of hitting the character service or Kafka.
type Seams struct {
	LoadCaster func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (int16, int16, error)
	EmitCreate func(l logrus.FieldLogger, ctx context.Context, body mistmsg.CreateCommandBody) error
}

// DefaultLoadCaster returns the caster's (X, Y) from the character service.
var DefaultLoadCaster = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (int16, int16, error) {
	c, err := character.NewProcessor(l, ctx).GetById()(characterId)
	if err != nil {
		return 0, 0, err
	}
	return c.X(), c.Y(), nil
}

// DefaultEmitCreate publishes the CREATE command to atlas-maps.
var DefaultEmitCreate = func(l logrus.FieldLogger, ctx context.Context, body mistmsg.CreateCommandBody) error {
	return mist.NewProcessor(l, ctx).Create(body)
}

// Cast validates the effect, loads the caster, and emits CREATE.
//
// Every rejection returns nil and emits nothing: there is no MP or cooldown
// rollback path, by design (task-200 FR-3.2 / FR-6.5). By the time this runs
// the cost has already been charged -- by processAttack for the attack-cast
// skills, by UseSkill for the USE_SKILL ones.
func Cast(
	l logrus.FieldLogger,
	ctx context.Context,
	f field.Model,
	characterId uint32,
	skillId skill2.Id,
	skillLevel byte,
	e effect.Model,
	p Params,
	s Seams,
) error {
	duration := e.Duration()
	lt, rb := e.LT(), e.RB()

	if duration <= 0 {
		l.Warnf("%s: rejected cast by [%d] — no lifetime (effect duration %d ms).", p.SkillName, characterId, duration)
		return nil
	}
	// A mist that never ticks (PROTECTION) has no sub-tick floor to clear.
	if p.TickMs > 0 && int64(duration) < p.TickMs {
		l.Warnf("%s: rejected cast by [%d] — lifetime shorter than one tick (%d ms < %d ms).", p.SkillName, characterId, duration, p.TickMs)
		return nil
	}
	if rb.X() <= lt.X() || rb.Y() <= lt.Y() {
		l.Warnf("%s: rejected cast by [%d] — degenerate rectangle lt(%d,%d) rb(%d,%d).", p.SkillName, characterId, lt.X(), lt.Y(), rb.X(), rb.Y())
		return nil
	}
	if duration > MaxPlayerMistDurationMs {
		l.Warnf("%s: rejected cast by [%d] — implausible lifetime (%d ms > %d ms ceiling).", p.SkillName, characterId, duration, MaxPlayerMistDurationMs)
		return nil
	}

	x, y, err := s.LoadCaster(l, ctx, characterId)
	if err != nil {
		l.WithError(err).Errorf("%s: failed to load caster [%d]; no mist created.", p.SkillName, characterId)
		return nil
	}

	body := mistmsg.CreateCommandBody{
		WorldId:    f.WorldId(),
		ChannelId:  f.ChannelId(),
		MapId:      f.MapId(),
		Instance:   f.Instance(),
		OwnerType:  "CHARACTER",
		OwnerId:    characterId,
		TargetKind: p.TargetKind,
		EffectKind: p.EffectKind,
		OriginX:    x,
		OriginY:    y,
		LtX:        int16(lt.X()),
		LtY:        int16(lt.Y()),
		RbX:        int16(rb.X()),
		RbY:        int16(rb.Y()),
		Disease:    p.Disease,
		// Magnitude 0 is correct, not a shortcut: the POISON magnitude is
		// TARGET-derived, so only atlas-monsters can fill it in. It resolves
		// the value per monster at apply time as
		// ceil(maxHP/(70 - sourceSkillLevel)) capped at 32767
		// (monster.ResolvePoisonDamage), and that single value is both the
		// damage each tick applies and the magnitude the client renders its
		// own tick numbers from. Anything sent here would be overwritten.
		DiseaseValue: 0,
		// Per-target duration = the mist's lifetime. With no WZ `dotTime`,
		// this is the value that matches observable behaviour, and
		// atlas-monsters REPLACES a same-type status on re-apply, so a
		// monster inside the cloud simply has its expiry pushed forward.
		DiseaseDuration: int64(duration),
		Duration:        int64(duration),
		TickIntervalMs:  p.TickMs,
		RecoveryMp:      p.RecoveryMp,
		PartyMemberIds:  p.PartyMemberIds,
		// The WIRE skill id, deliberately -- not the resolved Identity. The
		// client compares this against its own WZ to pick the rendering arm
		// (CAffectedAreaPool::AffectedAreaAnimationCreated, v83 @0x431d50,
		// v95 @0x437515), so it must be the id that version binds. This is
		// the one place a raw wire id is the correct value.
		SourceSkillId:    uint32(skillId),
		SourceSkillLevel: uint32(skillLevel),
	}

	if err := s.EmitCreate(l, ctx, body); err != nil {
		l.WithError(err).Errorf("%s: failed to emit CREATE for character [%d].", p.SkillName, characterId)
		return nil
	}

	l.Infof("%s: character [%d] cast level [%d] at (%d,%d), rect lt(%d,%d) rb(%d,%d), lifetime %d ms.",
		p.SkillName, characterId, skillLevel, x, y, lt.X(), lt.Y(), rb.X(), rb.Y(), duration)
	return nil
}
