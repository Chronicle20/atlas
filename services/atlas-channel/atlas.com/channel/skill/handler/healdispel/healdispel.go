package healdispel

import (
	"context"
	"math"

	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/data/skill/effect"
	"atlas-channel/effective_stats"
	channelmap "atlas-channel/map"
	"atlas-channel/session"
	channelhandler "atlas-channel/skill/handler"
	socketHandler "atlas-channel/socket/handler"
	"atlas-channel/socket/writer"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/sirupsen/logrus"
)

func init() {
	channelhandler.Register(skill2.SuperGmHealDispelId, Apply)
}

// diseaseTypes is the exact atlas-buffs disease set (buffs/character/immunity.go)
// that GM Heal + Dispel purges. Sourced from libs/atlas-constants (DOM-21).
var diseaseTypes = []string{
	string(charconst.TemporaryStatTypeStun),
	string(charconst.TemporaryStatTypePoison),
	string(charconst.TemporaryStatTypeSeal),
	string(charconst.TemporaryStatTypeDarkness),
	string(charconst.TemporaryStatTypeWeaken),
	string(charconst.TemporaryStatTypeCurse),
	string(charconst.TemporaryStatTypeSeduce),
	string(charconst.TemporaryStatTypeConfuse),
	string(charconst.TemporaryStatTypeUndead),
	string(charconst.TemporaryStatTypeSlow),
	string(charconst.TemporaryStatTypeStopPortion),
}

// healDispelDeps holds HealDispel's collaborators as function seams so the
// core loop is unit-testable offline (no Kafka/REST/session). announceSelf and
// announceForeign take the caster level so the wiring can build the skill-use
// packets without re-loading the caster.
type healDispelDeps struct {
	loadCaster      func(characterId uint32) (character.Model, error)
	isGmHidden      func(characterId uint32) (bool, error)
	selectInMap     func(f field.Model) []channelhandler.PartyRecipient
	effectiveMax    func(f field.Model, characterId uint32) (maxHp uint32, maxMp uint32, err error)
	changeHP        func(f field.Model, characterId uint32, amount int16) error
	changeMP        func(f field.Model, characterId uint32, amount int16) error
	dispel          func(f field.Model, characterId uint32, types []string) error
	announceSelf    func(level byte) error
	announceForeign func(level byte) error
}

// ratioAmount returns floor(max * ratio) as an int, or 0 for a non-positive ratio.
func ratioAmount(max uint16, ratio float64) int {
	if ratio <= 0 {
		return 0
	}
	return int(math.Floor(float64(max) * ratio))
}

// clampRestore turns a computed restore into the int16 delta to apply, clamped
// to [0, max-current] (never past the effective cap) and to the int16 ceiling.
func clampRestore(restore int, current uint16, max uint16) int16 {
	if restore <= 0 {
		return 0
	}
	headroom := int(max) - int(current)
	if headroom <= 0 {
		return 0
	}
	if restore > headroom {
		restore = headroom
	}
	if restore > math.MaxInt16 {
		restore = math.MaxInt16
	}
	return int16(restore)
}

// effectiveMaxOrBase narrows an effective-stats max (uint32) into uint16,
// falling back to the recipient's base max when the upstream returned zero or
// out-of-range. Mirrors the Cleric Heal clamp idiom.
func effectiveMaxOrBase(effective uint32, base uint16) uint16 {
	if effective == 0 {
		return base
	}
	if effective > math.MaxUint16 {
		return math.MaxUint16
	}
	return uint16(effective)
}

// applyHealDispel is the tested core: gate, select recipients, restore HP/MP,
// dispel diseases, then broadcast. Per-recipient failures are logged and never
// abort the others. No experience is ever awarded (GM utility, not combat heal).
func applyHealDispel(l logrus.FieldLogger, f field.Model, characterId uint32, e effect.Model, d healDispelDeps) error {
	c, err := d.loadCaster(characterId)
	if err != nil {
		l.WithError(err).Errorf("Heal+Dispel: failed to load caster [%d].", characterId)
		return nil
	}
	if !job.IsA(c.JobId(), job.SuperGmId) {
		l.Warnf("Character [%d] cast SuperGM Heal+Dispel without SuperGM job; rejecting.", characterId)
		return nil
	}

	hidden, hErr := d.isGmHidden(characterId)
	if hErr != nil {
		l.WithError(hErr).Debugf("Heal+Dispel: unable to resolve hidden state for caster [%d]; treating as visible.", characterId)
		hidden = false
	}

	recipients := d.selectInMap(f)
	for _, r := range recipients {
		effMaxHpRaw, effMaxMpRaw, sErr := d.effectiveMax(f, r.Id())
		if sErr != nil {
			l.WithError(sErr).Debugf("Heal+Dispel: effective stats fetch failed for recipient [%d]; using base maxes.", r.Id())
		}
		maxHp := effectiveMaxOrBase(effMaxHpRaw, r.MaxHp())
		maxMp := effectiveMaxOrBase(effMaxMpRaw, r.MaxMp())

		if hpDelta := clampRestore(int(e.HP())+ratioAmount(maxHp, e.HpR()), r.Hp(), maxHp); hpDelta > 0 {
			if err := d.changeHP(f, r.Id(), hpDelta); err != nil {
				l.WithError(err).Errorf("Heal+Dispel: ChangeHP failed for recipient [%d].", r.Id())
			}
		}
		if mpDelta := clampRestore(int(e.MP())+ratioAmount(maxMp, e.MpR()), r.Mp(), maxMp); mpDelta > 0 {
			if err := d.changeMP(f, r.Id(), mpDelta); err != nil {
				l.WithError(err).Errorf("Heal+Dispel: ChangeMP failed for recipient [%d].", r.Id())
			}
		}
		if err := d.dispel(f, r.Id(), diseaseTypes); err != nil {
			l.WithError(err).Errorf("Heal+Dispel: dispel failed for recipient [%d].", r.Id())
		}
	}

	if err := d.announceSelf(c.Level()); err != nil {
		l.WithError(err).Debugf("Heal+Dispel: self skill-use announce failed for caster [%d].", characterId)
	}
	if !hidden {
		if err := d.announceForeign(c.Level()); err != nil {
			l.WithError(err).Debugf("Heal+Dispel: foreign skill-use announce failed for caster [%d].", characterId)
		}
	}
	return nil
}

// Apply is the registered Heal + Dispel handler. It builds production deps and
// delegates to applyHealDispel.
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
			cp := character.NewProcessor(l, ctx)
			bp := buff.NewProcessor(l, ctx)
			esp := effective_stats.NewProcessor(l, ctx)
			sp := session.NewProcessor(l, ctx)
			mp := channelmap.NewProcessor(l, ctx)

			d := healDispelDeps{
				loadCaster: func(id uint32) (character.Model, error) { return cp.GetById()(id) },
				isGmHidden: func(id uint32) (bool, error) {
					bs, err := bp.GetByCharacterId(id)
					if err != nil {
						return false, err
					}
					return buff.IsGmHidden(bs), nil
				},
				selectInMap: func(f field.Model) []channelhandler.PartyRecipient {
					return channelhandler.SelectAllCharactersInMap(l, ctx, f)
				},
				effectiveMax: func(f field.Model, id uint32) (uint32, uint32, error) {
					s, err := esp.GetByCharacterId(f.WorldId(), f.ChannelId(), id)
					return s.MaxHp, s.MaxMp, err
				},
				changeHP: cp.ChangeHP,
				changeMP: cp.ChangeMP,
				dispel:   func(f field.Model, id uint32, types []string) error { return bp.CancelByTypes(f, id, types) },
				announceSelf: func(level byte) error {
					return sp.IfPresentByCharacterId(f.Channel())(
						characterId,
						socketHandler.AnnounceSkillUse(l)(ctx)(wp)(info.SkillId(), level, info.SkillLevel()),
					)
				},
				announceForeign: func(level byte) error {
					return mp.ForOtherSessionsInMap(
						f, characterId,
						socketHandler.AnnounceForeignSkillUse(l)(ctx)(wp)(characterId, info.SkillId(), level, info.SkillLevel()),
					)
				},
			}
			return applyHealDispel(l, f, characterId, e, d)
		}
	}
}
