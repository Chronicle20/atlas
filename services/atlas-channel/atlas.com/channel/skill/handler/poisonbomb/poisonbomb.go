// Package poisonbomb implements the Night Walker Poison Bomb (14111006)
// cast: it places a server-side mist at the caster's feet that poisons every
// monster inside its rectangle until it expires.
package poisonbomb

import (
	"atlas-channel/data/skill/effect"
	"atlas-channel/skill/handler/mistcast"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/point"

	mistmsg "atlas-channel/kafka/message/mist"
	channelhandler "atlas-channel/skill/handler"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// Poison Bomb is ATTACK-delivered, so it registers on the attack-cast
// registry: `GET /api/data/skills/14111006` serves mobCount 6 and
// damage 104->220 on every version served, plus prop 0.51->0.80 pre-v95 --
// real attack nodes, not the reader's absent-node defaults (damage 100,
// attackCount 1, mobCount 1). Registering it on the use-skill registry would
// never fire AND would suppress its generic MP cost.
func init() {
	channelhandler.RegisterAttackCast(skill2.NightWalkerStage3PoisonBomb, Apply)
}

var (
	loadCaster = mistcast.DefaultLoadCaster
	emitCreate = mistcast.DefaultEmitCreate
)

// Apply is the Poison Bomb handler installed in the per-skill attack-cast
// registry.
//
// The applied status is POISON with a target-derived magnitude, sent as 0 and
// resolved per monster by atlas-monsters' ResolvePoisonDamage -- exactly as
// Poison Mist does (FR-6.2).
func Apply(l logrus.FieldLogger) func(ctx context.Context) func(
	wp writer.Producer,
	f field.Model,
	characterId uint32,
	skillId skill2.Id,
	skillLevel byte,
	e effect.Model,
	castOrigin *point.Model,
) error {
	return func(ctx context.Context) func(
		wp writer.Producer,
		f field.Model,
		characterId uint32,
		skillId skill2.Id,
		skillLevel byte,
		e effect.Model,
		castOrigin *point.Model,
	) error {
		return func(
			_ writer.Producer,
			f field.Model,
			characterId uint32,
			skillId skill2.Id,
			skillLevel byte,
			e effect.Model,
			castOrigin *point.Model,
		) error {
			return mistcast.Cast(l, ctx, f, characterId, skillId, skillLevel, e,
				mistcast.Params{
					SkillName:  "Poison Bomb",
					TargetKind: mistmsg.TargetKindMonster,
					EffectKind: mistmsg.EffectKindDamageOverTime,
					Disease:    "POISON",
					TickMs:     mistcast.PlayerMistTickIntervalMs,
					// Poison Bomb is THROWN: the cloud belongs where the bomb
					// landed, which the attack packet carries and processAttack
					// passes down here. Nil (no grenade block) falls back to the
					// caster's feet.
					Origin: castOrigin,
				},
				mistcast.Seams{LoadCaster: loadCaster, EmitCreate: emitCreate})
		}
	}
}
