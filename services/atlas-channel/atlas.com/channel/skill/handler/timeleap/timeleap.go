// Package timeleap implements the Buccaneer Time Leap (5121010) per-skill
// handler. Time Leap's entire effect is server-side: it clears every active
// skill cooldown — except Time Leap's own — for the caster and for in-range
// party members (reference: Cosmic StatEffect removeAllCooldownsExcept).
//
// By the time this handler runs, UseSkill has already charged MP and applied
// Time Leap's own cooldown via SET_COOLDOWN, and the socket handler
// broadcasts the cast effect after UseSkill returns — this handler adds ONLY
// the RESET_COOLDOWNS emission (PRD FR-4). WZ-verified: 5121010 has an LT/RB
// rectangle at every level and no statups, so the generic buff path never
// fires and the rect-based party selector is the correct recipient filter.
package timeleap

import (
	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	"atlas-channel/socket/writer"
	"context"

	skillproc "atlas-channel/character/skill"

	channelhandler "atlas-channel/skill/handler"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

func init() {
	channelhandler.Register(skill2.BuccaneerTimeLeapId, Apply)
}

// loadCaster returns the caster's (X, Y) — only the position is needed for
// the recipient rectangle. Test seam (pattern: mysticdoor).
var loadCaster = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (int16, int16, error) {
	c, err := character.NewProcessor(l, ctx).GetById()(characterId)
	if err != nil {
		return 0, 0, err
	}
	return c.X(), c.Y(), nil
}

// selectParty resolves in-range party members. Test seam.
var selectParty = func(l logrus.FieldLogger, ctx context.Context, f field.Model, casterId uint32, x, y int16, e effect.Model, bitmap byte) []channelhandler.PartyRecipient {
	return channelhandler.SelectInRangePartyMembers(l, ctx, f, casterId, x, y, e, bitmap)
}

// emitReset sends one RESET_COOLDOWNS command for the recipient. Test seam.
var emitReset = func(l logrus.FieldLogger, ctx context.Context, transactionId uuid.UUID, f field.Model, exceptSkillIds []uint32, sourceSkillId uint32, characterId uint32) error {
	return skillproc.NewProcessor(l, ctx).ResetCooldowns(transactionId, f, exceptSkillIds, sourceSkillId)(characterId)
}

// Apply is the Time Leap handler installed in the per-skill registry.
//
// Lifecycle:
//  1. Load caster (X, Y). On failure: log, skip the reset entirely — the
//     cast continues (per-step failures never abort the cast, heal policy).
//  2. Resolve in-range party members (missing rect or bitmap 0 → empty →
//     caster-only; FR-5).
//  3. Emit one RESET_COOLDOWNS per recipient (caster + members) with a
//     single per-cast transactionId and exceptSkillIds=[5121010] — every
//     recipient keeps their own Time Leap cooldown (FR-3). Per-recipient
//     emission failures are logged and do not abort remaining recipients.
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
			x, y, err := loadCaster(l, ctx, characterId)
			if err != nil {
				l.WithError(err).Errorf("Time Leap: failed to load caster [%d].", characterId)
				return nil
			}

			channelhandler.WarnIfMissingRectangle(skill2.Id(info.SkillId()), info.SkillLevel(), e, func() {
				l.Warnf("Time Leap: skill effect [%d] level [%d] has no LT/RB rectangle — falling back to caster-only.", info.SkillId(), info.SkillLevel())
			})

			party := selectParty(l, ctx, f, characterId, x, y, e, info.AffectedPartyMemberBitmap())

			transactionId := uuid.New()
			except := []uint32{uint32(skill2.BuccaneerTimeLeapId)}
			source := uint32(skill2.BuccaneerTimeLeapId)

			if eErr := emitReset(l, ctx, transactionId, f, except, source, characterId); eErr != nil {
				l.WithError(eErr).Errorf("Time Leap: reset emission failed for caster [%d].", characterId)
			}
			for _, r := range party {
				if eErr := emitReset(l, ctx, transactionId, f, except, source, r.Id()); eErr != nil {
					l.WithError(eErr).Errorf("Time Leap: reset emission failed for recipient [%d] from caster [%d].", r.Id(), characterId)
				}
			}

			l.Debugf("Time Leap: caster=[%d] recipients=[%d] transaction=[%s].",
				characterId, 1+len(party), transactionId)
			return nil
		}
	}
}
