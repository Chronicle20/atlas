package heal

import (
	"context"
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

func init() {
	channelhandler.Register(skill2.ClericHealId, Apply)
}

// Apply is the Heal handler installed in the per-skill registry.
//
// Lifecycle:
//  1. Load caster character (X, Y, Hp, MaxHp, Level).
//  2. Load caster effective stats (INT, MagicAttack).
//  3. Resolve recipients: caster + in-range party members on the same
//     channel + map per the LT/RB rectangle and the affected-party
//     bitmap.
//  4. Compute the heal amount with a fresh [0.9, 1.1] variance roll.
//  5. Apply the HP delta to each recipient via character.ChangeHP.
//  6. Compute and award XP, gated by OQ-1 (skip when sole recipient
//     and no AffectedMobIds).
//  7. Broadcast CharacterEffect to caster + CharacterEffectForeign to
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

			stats, sErr := effective_stats.NewProcessor(l, ctx).GetByCharacterId(f.WorldId(), f.ChannelId(), characterId)
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
				MaxHp:    c.MaxHp(),
				IsCaster: true,
			}
			recipients := selectRecipients(caster, party)

			variance := 0.9 + rand.Float64()*0.2
			perTarget := HealAmount(
				e.HP(),
				int(stats.MagicAttack),
				int(stats.Intelligence),
				len(recipients),
				variance,
			)

			for _, r := range recipients {
				if hpErr := cp.ChangeHP(f, r.Id, perTarget); hpErr != nil {
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
