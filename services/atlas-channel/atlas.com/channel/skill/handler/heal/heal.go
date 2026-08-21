package heal

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/data/skill/effect"
	"atlas-channel/effective_stats"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"math"
	"math/rand"

	character2 "atlas-channel/kafka/message/character"
	channelmap "atlas-channel/map"

	channelhandler "atlas-channel/skill/handler"
	socketHandler "atlas-channel/socket/handler"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

// effectiveMaxHpOrBase narrows the effective MaxHp from
// atlas-effective-stats into the uint16 range used by the recipient
// snapshot, falling back to the recipient's base MaxHp when the
// upstream returned zero or out-of-range. Mirrors the defensive
// strategy in atlas-character's resolveEffectiveMax.
func effectiveMaxHpOrBase(effective uint32, base uint16) uint16 {
	if effective == 0 {
		return base
	}
	if effective > math.MaxUint16 {
		return math.MaxUint16
	}
	return uint16(effective)
}

// loadCasterFunc is the caster-load seam tests can replace. Production
// delegates to character.Processor.GetById.
var loadCasterFunc = func(cp character.Processor, characterId uint32) (character.Model, error) {
	return cp.GetById()(characterId)
}

// effectiveStatsFunc is the effective-stats seam tests can replace.
// Production delegates to effective_stats.Processor.GetByCharacterId.
var effectiveStatsFunc = func(esp effective_stats.Processor, worldId world.Id, channelId channel.Id, characterId uint32) (effective_stats.RestModel, error) {
	return esp.GetByCharacterId(worldId, channelId, characterId)
}

// selectPartyMembersFunc is the party-selection seam tests can replace.
// Production delegates to channelhandler.SelectInRangePartyMembers.
var selectPartyMembersFunc = channelhandler.SelectInRangePartyMembers

// varianceFunc is the [0.9, 1.1] heal roll, seamed so a cast's arithmetic is
// pinnable end-to-end.
var varianceFunc = func() float64 {
	return 0.9 + rand.Float64()*0.2
}

// casterZombifiedFunc is the zombify-state seam tests can replace. Production
// drains the caster's buffs from atlas-buffs and applies buff.IsZombified.
// Per task-256 FR-3 a failed read resolves to false (not zombified): a
// buff-service fault must never turn a Cleric's Heal into party-wide damage.
var casterZombifiedFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) bool {
	bs, err := buff.NewProcessor(l, ctx).GetByCharacterId(characterId)
	if err != nil {
		l.WithError(err).Warnf("Heal: buff read failed for caster [%d]; treating as not zombified.", characterId)
		return false
	}
	return buff.IsZombified(bs)
}

// changeHpFunc is the HP-change seam tests can replace. Production
// delegates to character.Processor.ChangeHP.
var changeHpFunc = func(cp character.Processor, f field.Model, characterId uint32, amount int16) error {
	return cp.ChangeHP(f, characterId, amount)
}

// awardExperienceFunc is the experience-award seam tests can replace.
// Production delegates to character.Processor.AwardExperience.
var awardExperienceFunc = func(cp character.Processor, f field.Model, characterId uint32, distributions []character2.ExperienceDistributions, showEffect bool) error {
	return cp.AwardExperience(f, characterId, distributions, showEffect)
}

// announceCastFunc broadcasts the cast to the caster and to every other
// session in the map. Seamed as one unit because both announcements are
// unconditional on every cast, negated or not (task-256 FR-16).
var announceCastFunc = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, f field.Model, characterId uint32, casterLevel byte, skillId uint32, skillLevel byte) {
	sp := session.NewProcessor(l, ctx)
	_ = sp.IfPresentByCharacterId(f.Channel())(
		characterId,
		socketHandler.AnnounceSkillUse(l)(ctx)(wp)(skillId, casterLevel, skillLevel),
	)
	_ = channelmap.NewProcessor(l, ctx).ForOtherSessionsInMap(
		f, characterId,
		socketHandler.AnnounceForeignSkillUse(l)(ctx)(wp)(characterId, skillId, casterLevel, skillLevel),
	)
}

func init() {
	channelhandler.Register(skill2.ClericHeal, Apply)
}

// Apply is the Heal handler installed in the per-skill registry.
//
// Lifecycle:
//  1. Load caster character (X, Y, Hp, MaxHp, Level).
//  2. Load caster effective stats (INT, MagicAttack, MaxHp).
//  3. Resolve recipients: caster + in-range party members on the same
//     channel + map per the LT/RB rectangle and the affected-party
//     bitmap.
//  4. Hydrate each recipient's MaxHp from atlas-effective-stats so the
//     subsequent clamp uses the player's actual cap, not the base
//     character record (which omits gear / buff bonuses).
//  5. Compute the heal amount with a fresh [0.9, 1.1] variance roll.
//  6. Per recipient: clamp delta to (effective MaxHp - current Hp) via
//     appliedPerRecipient, then call character.ChangeHP with the
//     clamped value. This prevents pushing Hp past MaxHp and tripping
//     atlas-character's enforceBounds saturation logic.
//  7. Compute and award XP from the same applied amounts (gated by
//     OQ-1: skip when sole recipient and no AffectedMobIds).
//  8. Broadcast CharacterEffect to caster + CharacterEffectForeign to
//     same-map sessions.
//
// Per-step failures are logged but do not abort the cast.
func Apply(l logrus.FieldLogger) func(ctx context.Context) func(
	wp writer.Producer,
	f field.Model, characterId uint32,
	info packetmodel.SkillUsageInfo, e effect.Model,
) error {
	return func(ctx context.Context) func(
		wp writer.Producer,
		f field.Model, characterId uint32,
		info packetmodel.SkillUsageInfo, e effect.Model,
	) error {
		return func(
			wp writer.Producer,
			f field.Model, characterId uint32,
			info packetmodel.SkillUsageInfo, e effect.Model,
		) error {
			cp := character.NewProcessor(l, ctx)
			c, err := loadCasterFunc(cp, characterId)
			if err != nil {
				l.WithError(err).Errorf("Heal: failed to load caster [%d].", characterId)
				return nil
			}

			esp := effective_stats.NewProcessor(l, ctx)
			stats, sErr := effectiveStatsFunc(esp, f.WorldId(), f.ChannelId(), characterId)
			if sErr != nil {
				l.WithError(sErr).Warnf("Heal: failed to load effective stats for caster [%d]; falling back to base character INT.", characterId)
				stats = effective_stats.RestModel{Intelligence: uint32(c.Intelligence())}
			}

			channelhandler.WarnIfMissingRectangle(skill2.Id(info.SkillId()), info.SkillLevel(), e, func() {
				l.Warnf("Heal: skill effect [%d] level [%d] has no LT/RB rectangle — falling back to caster-only.", info.SkillId(), info.SkillLevel())
			})

			party := selectPartyMembersFunc(l, ctx, f, characterId, c.X(), c.Y(), e, info.AffectedPartyMemberBitmap())
			caster := recipient{
				Id:       characterId,
				X:        c.X(),
				Y:        c.Y(),
				Hp:       c.Hp(),
				MaxHp:    effectiveMaxHpOrBase(stats.MaxHp, c.MaxHp()),
				IsCaster: true,
			}
			recipients := selectRecipients(caster, party)

			// Hydrate each non-caster recipient's MaxHp with their effective
			// stats so the per-recipient clamp uses the player's true cap.
			// Caster's MaxHp is already populated from the stats fetch above.
			for i := range recipients {
				if recipients[i].IsCaster {
					continue
				}
				rs, rErr := effectiveStatsFunc(esp, f.WorldId(), f.ChannelId(), recipients[i].Id)
				if rErr != nil {
					l.WithError(rErr).Debugf("Heal: effective stats fetch failed for recipient [%d]; using base MaxHp [%d].", recipients[i].Id, recipients[i].MaxHp)
					continue
				}
				recipients[i].MaxHp = effectiveMaxHpOrBase(rs.MaxHp, recipients[i].MaxHp)
			}

			// Resolved once, before the recipient loop and after the MaxHp
			// hydration above -- never per-recipient (FR-11).
			zombified := casterZombifiedFunc(l, ctx, characterId)

			variance := varianceFunc()
			perTarget := HealAmount(
				e.HP(),
				int(stats.MagicAttack),
				int(stats.Intelligence),
				len(recipients),
				variance,
			)

			for _, r := range recipients {
				delta := healDelta(perTarget, r, zombified)
				if delta == 0 {
					continue
				}
				if hpErr := changeHpFunc(cp, f, r.Id, delta); hpErr != nil {
					l.WithError(hpErr).Errorf("Heal: ChangeHP failed for recipient [%d] from caster [%d].", r.Id, characterId)
				}
			}

			// XP gate: skip when sole recipient AND no undead targets in this cast.
			// Also skipped entirely on a negated cast -- HealXp derives from the
			// applied heal, and a zombified cast heals nobody (task-256 FR-15).
			if !zombified && !(len(recipients) == 1 && len(info.AffectedMobIds()) == 0) {
				xp := HealXp(perTarget, recipients, info.SkillLevel())
				if xp > 0 {
					if xpErr := awardExperienceFunc(cp, f, characterId, []character2.ExperienceDistributions{{
						ExperienceType: character2.ExperienceDistributionTypeWhite,
						Amount:         xp,
					}}, false); xpErr != nil {
						l.WithError(xpErr).Errorf("Heal: AwardExperience failed for caster [%d].", characterId)
					}
				}
			}

			announceCastFunc(l, ctx, wp, f, characterId, c.Level(), info.SkillId(), info.SkillLevel())

			l.Debugf("Heal: caster=[%d] level=[%d] recipients=[%d] perTarget=[%d] zombified=[%t].",
				characterId, info.SkillLevel(), len(recipients), perTarget, zombified)

			return nil
		}
	}
}
