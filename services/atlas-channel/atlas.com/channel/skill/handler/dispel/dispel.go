package dispel

import (
	"atlas-channel/character/buff"
	"atlas-channel/data/skill/effect"
	"atlas-channel/socket/writer"
	"context"
	"math/rand"

	channelhandler "atlas-channel/skill/handler"

	"github.com/sirupsen/logrus"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

func init() {
	// Identity, not wire id: registry.go is keyed on skill2.Identity and
	// UseSkill resolves the incoming wire id through
	// constants.For(...).Skill.Resolve before Lookup (task-187). One
	// registration covers all provisioned versions.
	channelhandler.Register(skill2.PriestDispel, Apply)
}

// dispellableStatTypes is the exact Dispel cure set (PRD FR-5, Cosmic
// Character.dispelDebuffs parity). []string to match the shared
// buff.Processor.CancelByTypes signature (healdispel.diseaseTypes precedent).
// STUN / SEDUCE / CONFUSE / UNDEAD / STOP_PORTION / STOP_MOTION / FEAR are
// intentionally excluded -- atlas-monsters can inflict them, but they are
// cure-all (purgeDebuffs) semantics owned by SuperGM Heal+Dispel.
var dispellableStatTypes = []string{
	string(charconst.TemporaryStatTypeCurse),
	string(charconst.TemporaryStatTypeDarkness),
	string(charconst.TemporaryStatTypePoison),
	string(charconst.TemporaryStatTypeSeal),
	string(charconst.TemporaryStatTypeWeaken),
	string(charconst.TemporaryStatTypeSlow),
}

// selectPartyMembersFunc is the party-selection seam tests can replace.
// Production delegates to channelhandler.SelectPartyMembersInMap.
var selectPartyMembersFunc = channelhandler.SelectPartyMembersInMap

// propRollFunc gates per-recipient cure by the skill's prop value. Mirrors
// the unexported propRollFunc in skill/handler/common.go exactly.
// e.Prop() is pre-normalized 0.0-1.0 -- no /100.
var propRollFunc = func(prop float64) bool {
	if prop <= 0 {
		return false
	}
	if prop >= 1 {
		return true
	}
	return rand.Float64() <= prop
}

// cancelByTypesFunc is the buff-cancel seam tests can replace. Production
// delegates to buff.Processor.CancelByTypes.
var cancelByTypesFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, types []string) error {
	return buff.NewProcessor(l, ctx).CancelByTypes(f, characterId, types)
}

// Apply is the registered Priest Dispel handler. It cures the exact
// dispellableStatTypes set on the caster and on the bitmap-selected in-map
// party members, rolling the skill's prop per recipient. Per-recipient
// failures are logged and never abort the others. Always returns nil.
func Apply(l logrus.FieldLogger) func(ctx context.Context) func(
	wp writer.Producer, f field.Model, characterId uint32,
	info packetmodel.SkillUsageInfo, e effect.Model,
) error {
	return func(ctx context.Context) func(
		wp writer.Producer, f field.Model, characterId uint32,
		info packetmodel.SkillUsageInfo, e effect.Model,
	) error {
		return func(
			wp writer.Producer, f field.Model, characterId uint32,
			info packetmodel.SkillUsageInfo, e effect.Model,
		) error {
			bitmap := info.AffectedPartyMemberBitmap()
			members := selectPartyMembersFunc(l, ctx, f, characterId, bitmap)

			recipients := make([]uint32, 0, len(members)+1)
			recipients = append(recipients, characterId)
			for _, m := range members {
				recipients = append(recipients, m.Id())
			}

			var curesEmitted, propSkipped int
			for _, recipientId := range recipients {
				if !propRollFunc(e.Prop()) {
					propSkipped++
					continue
				}
				if err := cancelByTypesFunc(l, ctx, f, recipientId, dispellableStatTypes); err != nil {
					l.WithError(err).Errorf("Dispel: CancelByTypes failed for recipient [%d].", recipientId)
					continue
				}
				curesEmitted++
			}

			l.WithFields(logrus.Fields{
				"caster":              characterId,
				"skill_id":            info.SkillId(),
				"skill_level":         info.SkillLevel(),
				"bitmap":              bitmap,
				"recipients_selected": len(recipients),
				"cures_emitted":       curesEmitted,
				"prop_skipped":        propSkipped,
			}).Debug("dispel_party_cure_summary")

			return nil
		}
	}
}
