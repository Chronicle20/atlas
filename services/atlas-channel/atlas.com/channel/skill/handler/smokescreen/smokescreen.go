// Package smokescreen implements the Shadower Smokescreen (4221006) cast: it
// places a server-side smoke cloud at the caster's feet that shields the
// caster and their online party members from damage while they stand inside
// it.
//
// The mist itself never ticks. atlas-maps holds it and expires it; the
// protection is evaluated in atlas-channel on the damage path
// (socket/handler/character_damage_smoke.go), which is where the client
// evaluates it too -- CUserLocal::SetDamaged consults
// CAffectedAreaPool::IsSmokeAreaByPoint and, on a hit, jumps straight to the
// function epilogue before the miss roll, Power Guard, Meso Guard, Achilles
// and Magic Guard, sending no damage packet at all.
package smokescreen

import (
	"atlas-channel/data/skill/effect"
	"atlas-channel/skill/handler/mistcast"
	"atlas-channel/socket/writer"
	"context"

	mistmsg "atlas-channel/kafka/message/mist"
	channelhandler "atlas-channel/skill/handler"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// Smokescreen is USE_SKILL-delivered, so it registers on the use-skill
// registry: `GET /api/data/skills/4221006` serves damage 100, attackCount 1,
// mobCount 1 and prop 0 on all ten live tenants -- i.e. every attack node is
// ABSENT (those are the reader's defaults for a missing node,
// atlas-data skill/reader.go:197,268,270), not present-and-equal-to-default.
// UseSkill charges its MP consume and its 600s (360s at v95) cooldown before
// the handler lookup, so this handler charges nothing itself.
func init() {
	channelhandler.Register(skill2.ShadowerSmokescreen, Apply)
}

var (
	loadCaster = mistcast.DefaultLoadCaster
	emitCreate = mistcast.DefaultEmitCreate
)

// Apply is the Smokescreen handler installed in the per-skill use-skill
// registry.
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
			_ writer.Producer,
			f field.Model,
			characterId uint32,
			info packetmodel.SkillUsageInfo,
			e effect.Model,
		) error {
			return mistcast.Cast(l, ctx, f, characterId, skill2.Id(info.SkillId()), info.SkillLevel(), e,
				mistcast.Params{
					SkillName:  "Smokescreen",
					TargetKind: mistmsg.TargetKindCharacter,
					EffectKind: mistmsg.EffectKindProtection,
					// A protection mist has no per-tick effect: atlas-maps
					// only holds and expires it. TickMs 0 makes
					// Mist.ShouldTick false, so the tick returns before the
					// effect-kind switch is ever reached.
					TickMs: 0,
				},
				mistcast.Seams{LoadCaster: loadCaster, EmitCreate: emitCreate})
		}
	}
}
