// Package echoofhero implements the map-wide fan-out for the Echo of Hero
// family of skills (Beginner/Noblesse/Legend/Evan X005). Unlike the
// caster-only buffs UseSkill applies generically, Echo of Hero benefits
// every live-session character in the caster's field.
//
// This file holds the pure, offline-testable core (applyEchoOfHero) plus its
// seam struct (echoDeps). Task-162's follow-up task adds the init()
// registration and the production Apply wiring that builds echoDeps from
// real collaborators (character/buff/map processors) -- deliberately not
// present here.
package echoofhero

import (
	"atlas-channel/character/buff"
	"atlas-channel/data/skill/effect"
	"atlas-channel/socket/writer"
	"context"
	"sort"

	channelhandler "atlas-channel/skill/handler"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	model "github.com/Chronicle20/atlas/libs/atlas-model/model"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

func init() {
	channelhandler.Register(skill2.BeginnerEchoOfHero, Apply)
	channelhandler.Register(skill2.NoblesseEchoOfHero, Apply)
	channelhandler.Register(skill2.LegendEchoOfHero, Apply)
	channelhandler.Register(skill2.EvanEchoOfHero, Apply)
}

// echoDeps holds Echo of Hero's fan-out collaborators as function seams so
// the core loop is unit-testable offline (no Kafka/REST/session).
// applyBuff is the already-constructed buff operator for this cast (the same
// operator the generic UseSkill step used to buff the caster) -- this
// package never builds it itself.
type echoDeps struct {
	selectInMap func(f field.Model) []channelhandler.PartyRecipient
	isGmHidden  func(characterId uint32) (bool, error)
	applyBuff   model.Operator[uint32]
}

// applyEchoOfHero is the tested core: fan the already-constructed buff
// operator out to every live-session character in the field except the
// caster (already buffed by the generic step, FR-1.1), dead characters
// (FR-2.2), and hidden GMs (FR-2.3). A per-recipient failure -- hidden-state
// lookup or the buff apply itself -- is logged and skipped; it never aborts
// the remaining recipients or fails the cast (FR-2.5).
func applyEchoOfHero(
	l logrus.FieldLogger, f field.Model, casterId uint32,
	info packetmodel.SkillUsageInfo, e effect.Model, d echoDeps,
) error {
	if e.Duration() <= 0 || len(e.StatUps()) == 0 {
		return nil
	}

	rs := d.selectInMap(f)
	sort.Slice(rs, func(i, j int) bool { return rs[i].Id() < rs[j].Id() })

	var applied, skippedCaster, skippedDead, skippedHidden, fetchFailures, applyFailures int

	for _, r := range rs {
		if r.Id() == casterId {
			skippedCaster++
			continue
		}
		if r.Hp() == 0 {
			skippedDead++
			continue
		}
		hidden, hErr := d.isGmHidden(r.Id())
		if hErr != nil {
			fetchFailures++
			l.WithError(hErr).Debugf("Echo of Hero: unable to resolve hidden state for recipient [%d]; skipping.", r.Id())
			continue
		}
		if hidden {
			skippedHidden++
			continue
		}
		if aErr := d.applyBuff(r.Id()); aErr != nil {
			applyFailures++
			l.WithError(aErr).Errorf("Echo of Hero: buff apply failed for recipient [%d].", r.Id())
			continue
		}
		applied++
	}

	l.WithFields(logrus.Fields{
		"caster":         casterId,
		"skill_id":       info.SkillId(),
		"skill_level":    info.SkillLevel(),
		"in_map":         len(rs),
		"applied":        applied,
		"skipped_caster": skippedCaster,
		"skipped_dead":   skippedDead,
		"skipped_hidden": skippedHidden,
		"fetch_failures": fetchFailures,
		"apply_failures": applyFailures,
	}).Debug("echo_of_hero_apply_summary")

	return nil
}

// Apply is the registered Echo of Hero handler for all four identities
// (Beginner/Noblesse/Legend/Evan). It builds production deps and delegates to
// applyEchoOfHero. applyBuff reuses the same buff operator the generic
// UseSkill step already constructed to buff the caster (e.StatUps(), not a
// rewritten set -- the Shadow Stars rewrite in UseSkill applies to that skill
// only), so every recipient in the fan-out gets an identical buff to the
// caster's.
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
			bp := buff.NewProcessor(l, ctx)

			d := echoDeps{
				selectInMap: func(f field.Model) []channelhandler.PartyRecipient {
					return channelhandler.SelectAllCharactersInMap(l, ctx, f)
				},
				isGmHidden: func(id uint32) (bool, error) {
					bs, err := bp.GetByCharacterId(id)
					if err != nil {
						return false, err
					}
					return buff.IsGmHidden(ctx, bs), nil
				},
				applyBuff: bp.Apply(f, characterId, int32(info.SkillId()), info.SkillLevel(), e.Duration(), e.StatUps()),
			}
			return applyEchoOfHero(l, f, characterId, info, e, d)
		}
	}
}
