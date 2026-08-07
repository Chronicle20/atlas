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
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

func init() {
	channelhandler.Register(skill2.FirePoisonMagicianPoisonMist, Apply)
}

// PlayerMistTickIntervalMs is the per-tick cadence of a player-cast mist.
//
// It is a constant, not a WZ value: the `dotInterval` node does not exist in
// any provisioned Skill.wz (task-200 design §2.1). 1 Hz is already the
// de-facto DoT cadence on both ends of this contract -- the monster
// AREA_POISON producer hard-codes TickIntervalMs: 1000, and atlas-monsters'
// APPLY_STATUS consumer independently defaults a POISON/VENOM tick to 1000ms
// when the command omits one. This makes it explicit rather than relying on
// the consumer's fallback.
//
// Known tuning point: atlas-monsters replaces a same-type status on re-apply,
// minting a fresh lastTick, so a 1000ms re-apply against a 1s DoT cadence can
// under-count ticks. If observed damage is starved, raise this above the DoT
// cadence -- a one-constant change (design §4.4).
const PlayerMistTickIntervalMs int64 = 1000

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

// Apply is the Poison Mist handler installed in the per-skill registry.
//
// By the time it runs, UseSkill has already charged MP and applied the
// cooldown. Every rejection below returns nil and emits nothing: there is no
// MP or cooldown rollback path, by design (FR-3.2 / FR-6.5).
func Apply(l logrus.FieldLogger) func(ctx context.Context) func(
	wp writer.Producer,
	f field.Model,
	characterId uint32,
	info packetmodel.SkillUsageInfo,
	e effect.Model,
) error {
	return func(ctx context.Context) func(
		wp writer.Producer,
		f field.Model,
		characterId uint32,
		info packetmodel.SkillUsageInfo,
		e effect.Model,
	) error {
		return func(
			wp writer.Producer,
			f field.Model,
			characterId uint32,
			info packetmodel.SkillUsageInfo,
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
				// Magnitude 0 is correct, not a shortcut: atlas-monsters
				// computes poison damage as maxHP/(70 - sourceSkillLevel) and
				// never reads the POISON magnitude (VENOM is the status that
				// does). A non-zero value here would be dead payload.
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
				SourceSkillId:    uint32(info.SkillId()),
				SourceSkillLevel: uint32(info.SkillLevel()),
			}

			if err := emitCreate(l, ctx, body); err != nil {
				l.WithError(err).Errorf("Poison Mist: failed to emit CREATE for character [%d].", characterId)
				return nil
			}

			l.Infof("Poison Mist: character [%d] cast level [%d] at (%d,%d), rect lt(%d,%d) rb(%d,%d), lifetime %d ms.",
				characterId, info.SkillLevel(), x, y, lt.X(), lt.Y(), rb.X(), rb.Y(), duration)
			return nil
		}
	}
}
