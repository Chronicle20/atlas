// Package chakra is the per-skill USE_SKILL handler for Chief Bandit Chakra
// (4211001). The recovery window it consumes is opened on the skill-prepare
// packet (socket/handler/character_skill_prepare.go) and lives in
// atlas-channel/character/chakra.
package chakra

import (
	"atlas-channel/character"
	chakrastate "atlas-channel/character/chakra"
	"atlas-channel/data/skill/effect"
	"atlas-channel/effective_stats"
	channelhandler "atlas-channel/skill/handler"
	"atlas-channel/socket/writer"
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func init() {
	channelhandler.Register(skill2.ChiefBanditChakra, Apply)
}

// healDelta computes the HP Chakra restores on this completion, clamped to
// the caster's missing HP.
//
// The recovery rate comes from the WINDOW's snapshot, not from the effect
// the USE_SKILL packet resolved: the window captured `y` at prepare time
// from the caster's real skill-book level, so a client that lies about its
// level on the second packet cannot inflate the heal.
func healDelta(entry chakrastate.Entry, luck uint32, hp uint16, maxHp uint16) int16 {
	return chakrastate.Applied(
		chakrastate.Recovery(chakrastate.Base(luck), entry.Y),
		hp,
		maxHp,
	)
}

// Apply is the Chakra handler installed in the per-skill registry.
//
// It runs at the COMPLETION of the recovery window — the client sends this
// USE_SKILL at the end of the 1500 ms prepare animation (design §2), which
// is why the heal lands here and not at keypress (PRD FR-3.4).
//
// It deliberately does NOT:
//   - charge MP or apply cooldown — the generic UseSkill block owns both and
//     has already run by the time this is dispatched (PRD FR-8.2/8.3);
//   - award experience — Chakra is self-only (PRD FR-8.4);
//   - broadcast the cast effect — character_skill_use.go already announces
//     AnnounceSkillUse and AnnounceForeignSkillUse unconditionally after
//     UseSkill returns, so re-announcing here would send it twice.
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
			t := tenant.MustFromContext(ctx)
			reg := chakrastate.GetRegistry()

			entry, ok := reg.Get(t, characterId, time.Now())
			if !ok {
				// The pre-cost gate in character_skill_use.go rejects this
				// case before UseSkill runs; reaching it here means the
				// window expired between the two checks.
				l.Debugf("Chakra: no open recovery window for character [%d] at completion; no heal applied.", characterId)
				return nil
			}
			// The window is consumed whether or not the heal lands, so a
			// failed lookup cannot leave a stale damage factor behind.
			defer reg.Clear(t, characterId)

			cp := character.NewProcessor(l, ctx)
			c, err := cp.GetById()(characterId)
			if err != nil {
				l.WithError(err).Errorf("Chakra: failed to load caster [%d]; no heal applied.", characterId)
				return nil
			}

			luck := uint32(c.Luck())
			maxHp := c.MaxHp()
			stats, sErr := effective_stats.NewProcessor(l, ctx).GetByCharacterId(f.WorldId(), f.ChannelId(), characterId)
			if sErr != nil {
				l.WithError(sErr).Warnf("Chakra: effective stats unavailable for caster [%d]; falling back to base LUK and base max hp.", characterId)
			} else {
				luck = stats.Luck
				maxHp = chakrastate.EffectiveMaxHpOrBase(stats.MaxHp, c.MaxHp())
			}

			delta := healDelta(entry, luck, c.Hp(), maxHp)
			if delta == 0 {
				l.Debugf("Chakra: caster [%d] completed at level [%d] with no HP headroom (hp [%d] of [%d]); no ChangeHP emitted.", characterId, entry.SkillLevel, c.Hp(), maxHp)
				return nil
			}

			if hpErr := cp.ChangeHP(f, characterId, delta); hpErr != nil {
				l.WithError(hpErr).Errorf("Chakra: ChangeHP failed for caster [%d].", characterId)
				return nil
			}

			l.Debugf("Chakra: caster [%d] level [%d] restored [%d] hp (luck [%d], y [%d]%%, hp [%d] of [%d]).",
				characterId, entry.SkillLevel, delta, luck, entry.Y, c.Hp(), maxHp)
			return nil
		}
	}
}
