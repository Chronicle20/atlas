// Package poisonmist implements the Fire/Poison Mage Poison Mist (2111003)
// cast: it places a server-side mist at the caster's feet that poisons every
// monster inside its rectangle until it expires.
package poisonmist

import (
	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	"atlas-channel/mist"
	"atlas-channel/socket/writer"
	"context"

	mistmsg "atlas-channel/kafka/message/mist"
	channelhandler "atlas-channel/skill/handler"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// Poison Mist is registered on the ATTACK-cast registry, not the use-skill
// one. The client delivers 2111003 on a magic-attack packet (opcode 0x2E at
// GMS v83) because the skill carries `damage`/`attackCount`/`mobCount` in
// Skill.wz -- verified live: `GET /api/data/skills/2111003` returns
// damage 100, attackCount 1, mobCount 1, prop 0.41. It never arrives on
// USE_SKILL, so a channelhandler.Register here would never fire (and would
// additionally suppress the generic MP cost -- see AttackCastHandler's doc).
func init() {
	channelhandler.RegisterAttackCast(skill2.FirePoisonMagicianPoisonMist, Apply)
}

// PlayerMistTickIntervalMs is the RE-APPLY cadence of a player-cast mist --
// how often atlas-maps re-issues APPLY_STATUS(POISON) to every monster still
// standing in the cloud (mist.Mist.ShouldTick, mist/model.go:216-223). It is
// NOT the DoT damage cadence and does NOT flow into atlas-monsters' DoT gate.
//
// atlas-maps sends a SEPARATE, strictly smaller DoT tick interval on every
// APPLY_STATUS command (monsterDotTickIntervalMs,
// services/atlas-maps/atlas.com/maps/tasks/mist_tick.go, currently 1000ms).
// atlas-monsters' StatusEffect.ShouldTick gates actual damage on
// `since(lastTick) >= tickInterval`
// (services/atlas-monsters/atlas.com/monsters/monster/status.go:129-134).
//
// It must be an interval, not a WZ value: the `dotInterval` node does not
// exist in any provisioned Skill.wz (task-200 design §2.1).
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
// lifetime. The largest legitimate `time` for 2111003 across the provisioned
// corpus is 40s at level 30, so this 5-minute ceiling is 7.5x the largest real
// value and can only fire on corrupt or mis-scaled data.
//
// This is deliberately NOT atlas-monsters' 60s MistDurationCapMs. A clamp
// would desynchronise the client, which computes its own
// tEnd = tStart + 1000*SKILLLEVELDATA::tTime from its own WZ (v83 @0x43200f,
// v95 @0x437c95) and would keep rendering a mist the server stopped ticking.
const MaxPlayerMistDurationMs int32 = 300_000

// loadCaster returns the caster's (X, Y) position from the character service.
// Package-level var so tests can stub it.
var loadCaster = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (int16, int16, error) {
	c, err := character.NewProcessor(l, ctx).GetById()(characterId)
	if err != nil {
		return 0, 0, err
	}
	return c.X(), c.Y(), nil
}

// emitCreate publishes the CREATE command to atlas-maps. Package-level var so
// tests can record instead of producing.
var emitCreate = func(l logrus.FieldLogger, ctx context.Context, body mistmsg.CreateCommandBody) error {
	return mist.NewProcessor(l, ctx).Create(body)
}

// Apply is the Poison Mist handler installed in the per-skill attack-cast
// registry.
//
// By the time it runs, processAttack has already charged MP, applied the
// direct magic-attack damage, and broadcast the attack. Every rejection below
// returns nil and emits nothing: there is no MP or cooldown rollback path, by
// design (FR-3.2 / FR-6.5).
func Apply(l logrus.FieldLogger) func(ctx context.Context) func(
	wp writer.Producer,
	f field.Model,
	characterId uint32,
	skillId skill2.Id,
	skillLevel byte,
	e effect.Model,
) error {
	return func(ctx context.Context) func(
		wp writer.Producer,
		f field.Model,
		characterId uint32,
		skillId skill2.Id,
		skillLevel byte,
		e effect.Model,
	) error {
		return func(
			wp writer.Producer,
			f field.Model,
			characterId uint32,
			skillId skill2.Id,
			skillLevel byte,
			e effect.Model,
		) error {
			duration := e.Duration()
			lt, rb := e.LT(), e.RB()

			if duration <= 0 {
				l.Warnf("Poison Mist: rejected cast by [%d] — no lifetime (effect duration %d ms).", characterId, duration)
				return nil
			}
			if int64(duration) < PlayerMistTickIntervalMs {
				l.Warnf("Poison Mist: rejected cast by [%d] — lifetime shorter than one tick (%d ms < %d ms).", characterId, duration, PlayerMistTickIntervalMs)
				return nil
			}
			if rb.X() <= lt.X() || rb.Y() <= lt.Y() {
				l.Warnf("Poison Mist: rejected cast by [%d] — degenerate rectangle lt(%d,%d) rb(%d,%d).", characterId, lt.X(), lt.Y(), rb.X(), rb.Y())
				return nil
			}
			if duration > MaxPlayerMistDurationMs {
				l.Warnf("Poison Mist: rejected cast by [%d] — implausible lifetime (%d ms > %d ms ceiling).", characterId, duration, MaxPlayerMistDurationMs)
				return nil
			}

			x, y, err := loadCaster(l, ctx, characterId)
			if err != nil {
				l.WithError(err).Errorf("Poison Mist: failed to load caster [%d]; no mist created.", characterId)
				return nil
			}

			body := mistmsg.CreateCommandBody{
				WorldId:   f.WorldId(),
				ChannelId: f.ChannelId(),
				MapId:     f.MapId(),
				Instance:  f.Instance(),
				OwnerType: "CHARACTER",
				OwnerId:   characterId,
				// A player-cast mist targets MONSTERS with a damage-bearing
				// status, unlike the monster AREA_POISON mist which diseases
				// CHARACTERS.
				TargetKind: mistmsg.TargetKindMonster,
				EffectKind: mistmsg.EffectKindDamageOverTime,
				OriginX:    x,
				OriginY:    y,
				LtX:        int16(lt.X()),
				LtY:        int16(lt.Y()),
				RbX:        int16(rb.X()),
				RbY:        int16(rb.Y()),
				Disease:    "POISON",
				// Magnitude 0 is correct, not a shortcut: the POISON
				// magnitude is TARGET-derived, so only atlas-monsters can fill
				// it in. It resolves the value per monster at apply time as
				// ceil(maxHP/(70 - sourceSkillLevel)) capped at 32767
				// (monster.ResolvePoisonDamage), and that single value is both
				// the damage each tick applies and the magnitude the client
				// renders its own tick numbers from. Anything sent here would
				// be overwritten.
				DiseaseValue: 0,
				// Per-target duration = the mist's lifetime. With no WZ
				// `dotTime`, this is the value that matches the skill's
				// observable behavior, and atlas-monsters REPLACES a same-type
				// status on re-apply, so a monster inside the cloud simply has
				// its expiry pushed forward each tick (design D1a / §4.4).
				DiseaseDuration: int64(duration),
				Duration:        int64(duration),
				TickIntervalMs:  PlayerMistTickIntervalMs,
				// The WIRE skill id, deliberately -- not the resolved
				// Identity. The client compares this against its own WZ to
				// pick the rendering arm (v83 @0x431d50, v95 @0x437515), so it
				// must be the id that version binds. This is the one place a
				// raw wire id is the correct value.
				SourceSkillId:    uint32(skillId),
				SourceSkillLevel: uint32(skillLevel),
			}

			if err := emitCreate(l, ctx, body); err != nil {
				l.WithError(err).Errorf("Poison Mist: failed to emit CREATE for character [%d].", characterId)
				return nil
			}

			l.Infof("Poison Mist: character [%d] cast level [%d] at (%d,%d), rect lt(%d,%d) rb(%d,%d), lifetime %d ms.",
				characterId, skillLevel, x, y, lt.X(), lt.Y(), rb.X(), rb.Y(), duration)
			return nil
		}
	}
}
