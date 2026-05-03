package heal

import (
	"context"
	"math"
	"math/rand"

	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	"atlas-channel/effective_stats"
	character2 "atlas-channel/kafka/message/character"
	channelmap "atlas-channel/map"
	"atlas-channel/session"
	channelhandler "atlas-channel/skill/handler"
	socketHandler "atlas-channel/socket/handler"
	"atlas-channel/socket/writer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
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

func init() {
	channelhandler.Register(skill2.ClericHealId, Apply)
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
			c, err := cp.GetById()(characterId)
			if err != nil {
				l.WithError(err).Errorf("Heal: failed to load caster [%d].", characterId)
				return nil
			}

			esp := effective_stats.NewProcessor(l, ctx)
			stats, sErr := esp.GetByCharacterId(f.WorldId(), f.ChannelId(), characterId)
			if sErr != nil {
				l.WithError(sErr).Warnf("Heal: failed to load effective stats for caster [%d]; falling back to base character INT.", characterId)
				stats = effective_stats.RestModel{Intelligence: uint32(c.Intelligence())}
			}

			warnIfMissingRectangle(skill2.Id(info.SkillId()), info.SkillLevel(), e, func() {
				l.Warnf("Heal: skill effect [%d] level [%d] has no LT/RB rectangle — falling back to caster-only.", info.SkillId(), info.SkillLevel())
			})

			party := channelhandler.SelectInRangePartyMembers(l, ctx, f, characterId, c.X(), c.Y(), e, info.AffectedPartyMemberBitmap())
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
				rs, rErr := esp.GetByCharacterId(f.WorldId(), f.ChannelId(), recipients[i].Id)
				if rErr != nil {
					l.WithError(rErr).Debugf("Heal: effective stats fetch failed for recipient [%d]; using base MaxHp [%d].", recipients[i].Id, recipients[i].MaxHp)
					continue
				}
				recipients[i].MaxHp = effectiveMaxHpOrBase(rs.MaxHp, recipients[i].MaxHp)
			}

			variance := 0.9 + rand.Float64()*0.2
			perTarget := HealAmount(
				e.HP(),
				int(stats.MagicAttack),
				int(stats.Intelligence),
				len(recipients),
				variance,
			)

			for _, r := range recipients {
				delta := appliedPerRecipient(perTarget, r)
				if delta == 0 {
					continue
				}
				if hpErr := cp.ChangeHP(f, r.Id, delta); hpErr != nil {
					l.WithError(hpErr).Errorf("Heal: ChangeHP failed for recipient [%d] from caster [%d].", r.Id, characterId)
				}
			}

			// XP gate: skip when sole recipient AND no undead targets in this cast.
			if !(len(recipients) == 1 && len(info.AffectedMobIds()) == 0) {
				xp := HealXp(perTarget, recipients, info.SkillLevel())
				if xp > 0 {
					if xpErr := cp.AwardExperience(f, characterId, []character2.ExperienceDistributions{{
						ExperienceType: character2.ExperienceDistributionTypeWhite,
						Amount:         xp,
					}}, false); xpErr != nil {
						l.WithError(xpErr).Errorf("Heal: AwardExperience failed for caster [%d].", characterId)
					}
				}
			}

			sp := session.NewProcessor(l, ctx)
			_ = sp.IfPresentByCharacterId(f.Channel())(
				characterId,
				socketHandler.AnnounceSkillUse(l)(ctx)(wp)(info.SkillId(), c.Level(), info.SkillLevel()),
			)
			_ = channelmap.NewProcessor(l, ctx).ForOtherSessionsInMap(
				f, characterId,
				socketHandler.AnnounceForeignSkillUse(l)(ctx)(wp)(characterId, info.SkillId(), c.Level(), info.SkillLevel()),
			)

			l.Debugf("Heal: caster=[%d] level=[%d] recipients=[%d] perTarget=[%d].",
				characterId, info.SkillLevel(), len(recipients), perTarget)

			return nil
		}
	}
}
