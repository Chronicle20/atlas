// Package poisonmist implements the Fire/Poison Mage Poison Mist (2111003)
// cast: it places a server-side mist at the caster's feet that poisons every
// monster inside its rectangle until it expires.
package poisonmist

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

// loadCaster / emitCreate are this handler's copies of the mistcast seams.
// Package-level vars so tests can record instead of calling the character
// service and Kafka.
var (
	loadCaster = mistcast.DefaultLoadCaster
	emitCreate = mistcast.DefaultEmitCreate
)

// Apply is the Poison Mist handler installed in the per-skill attack-cast
// registry.
//
// By the time it runs, processAttack has already charged MP, applied the
// direct magic-attack damage, and broadcast the attack. Every rejection
// inside mistcast.Cast returns nil and emits nothing: there is no MP or
// cooldown rollback path, by design (FR-3.2 / FR-6.5).
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
					SkillName: "Poison Mist",
					// A player-cast mist targets MONSTERS with a
					// damage-bearing status, unlike the monster AREA_POISON
					// mist which diseases CHARACTERS.
					TargetKind: mistmsg.TargetKindMonster,
					EffectKind: mistmsg.EffectKindDamageOverTime,
					Disease:    "POISON",
					TickMs:     mistcast.PlayerMistTickIntervalMs,
					// The attack packet's own caster position. Reading it back
					// from atlas-character instead is asynchronous and anchored
					// the cloud where the caster used to be (task-218 #2).
					Origin: castOrigin,
				},
				mistcast.Seams{LoadCaster: loadCaster, EmitCreate: emitCreate})
		}
	}
}
