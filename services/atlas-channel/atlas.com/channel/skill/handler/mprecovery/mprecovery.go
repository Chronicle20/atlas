package mprecovery

import (
	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	"atlas-channel/socket/writer"
	"context"

	channelhandler "atlas-channel/skill/handler"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

func init() {
	channelhandler.Register(skill2.BrawlerMPRecovery, Apply)
}

// loadCaster returns the caster's max HP from the character service.
var loadCaster = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (uint16, error) {
	c, err := character.NewProcessor(l, ctx).GetById()(characterId)
	if err != nil {
		return 0, err
	}
	return c.MaxHp(), nil
}

// changeHP emits the HP-change command to atlas-character.
var changeHP = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, amount int16) error {
	return character.NewProcessor(l, ctx).ChangeHP(f, characterId, amount)
}

// changeMP emits the MP-change command to atlas-character.
var changeMP = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32, amount int16) error {
	return character.NewProcessor(l, ctx).ChangeMP(f, characterId, amount)
}

// Apply is the MP Recovery handler installed in the per-skill registry.
//
// By the time this runs, UseSkill has already applied the cooldown (5101005
// has no hpCon/mpCon/duration in WZ, so those generic branches are no-ops).
// The effect is entirely server-authoritative: lose MaxHP/x HP, gain
// hpLost*y/100 MP (WZ-verified v83 formula). Deliberately no low-HP guard — a
// caster at or below MaxHP/x current HP dies through atlas-character's
// existing ChangeHP 0-floor/death path (owner decision, task-151 PRD FR-3).
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
			maxHp, err := loadCaster(l, ctx, characterId)
			if err != nil {
				l.WithError(err).Errorf("MP Recovery: failed to load caster [%d].", characterId)
				return err
			}

			hpLost, mpGain := Amounts(maxHp, e.X(), e.Y())
			if hpLost == 0 {
				l.Warnf("MP Recovery: no HP cost for caster [%d] (maxHp=[%d] x=[%d]); skipping.",
					characterId, maxHp, e.X())
				return nil
			}

			if err := changeHP(l, ctx, f, characterId, -hpLost); err != nil {
				l.WithError(err).Errorf("MP Recovery: ChangeHP failed for caster [%d]; skipping MP gain.", characterId)
				return err
			}
			if mpGain > 0 {
				if err := changeMP(l, ctx, f, characterId, mpGain); err != nil {
					l.WithError(err).Errorf("MP Recovery: ChangeMP failed for caster [%d].", characterId)
					return err
				}
			}

			l.Debugf("MP Recovery: caster=[%d] level=[%d] hpLost=[%d] mpGain=[%d].",
				characterId, info.SkillLevel(), hpLost, mpGain)
			return nil
		}
	}
}
