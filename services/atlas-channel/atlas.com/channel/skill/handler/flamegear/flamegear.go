// Package flamegear implements the Blaze Wizard Flame Gear (12111005) cast:
// it places a server-side mist at the caster's feet that poisons every
// monster inside its rectangle until it expires.
package flamegear

import (
	"atlas-channel/data/skill/effect"
	"atlas-channel/skill/handler/mistcast"
	"atlas-channel/socket/writer"
	"context"

	mistmsg "atlas-channel/kafka/message/mist"
	channelhandler "atlas-channel/skill/handler"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// Flame Gear is ATTACK-delivered, so it registers on the attack-cast
// registry: `GET /api/data/skills/12111005` serves prop 0.51->0.80 on gms
// 72/79/83/84/87/92 and jms 185, and mobCount 8 / damage 124 at gms 95 --
// real attack nodes, not the reader's absent-node defaults (damage 100,
// attackCount 1, mobCount 1). Registering it on the use-skill registry would
// never fire AND would suppress its generic MP cost.
func init() {
	channelhandler.RegisterAttackCast(skill2.BlazeWizardStage3FlameGear, Apply)
}

var (
	loadCaster = mistcast.DefaultLoadCaster
	emitCreate = mistcast.DefaultEmitCreate
)

// Apply is the Flame Gear handler installed in the per-skill attack-cast
// registry.
//
// The applied status is POISON with a target-derived magnitude (sent as 0 and
// resolved per monster by atlas-monsters' ResolvePoisonDamage). This is
// WZ-derived, not an analogy with Poison Mist: `monsterStatus` is {} for
// 12111005 on every version served, atlas-monsters implements exactly two DoT
// statuses (POISON and the caster-magnitude VENOM), and the only caster-side
// magnitude candidate -- `dot` -- is 0 on six of the seven versions that bind
// the skill (task-218 design §1.4).
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
			_ *point.Model,
		) error {
			return mistcast.Cast(l, ctx, f, characterId, skillId, skillLevel, e,
				mistcast.Params{
					SkillName:  "Flame Gear",
					TargetKind: mistmsg.TargetKindMonster,
					EffectKind: mistmsg.EffectKindDamageOverTime,
					Disease:    "POISON",
					TickMs:     mistcast.PlayerMistTickIntervalMs,
				},
				mistcast.Seams{LoadCaster: loadCaster, EmitCreate: emitCreate})
		}
	}
}
