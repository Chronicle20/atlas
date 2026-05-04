package character

import (
	"atlas-effective-stats/external/buffs"
	"atlas-effective-stats/external/character"
	"atlas-effective-stats/external/data/equipment"
	skilldata "atlas-effective-stats/external/data/skill"
	"atlas-effective-stats/external/inventory"
	"atlas-effective-stats/external/skills"
	"atlas-effective-stats/stat"
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/sirupsen/logrus"
)

// IsInitialized checks if a character has been initialized
func IsInitialized(ctx context.Context, characterId uint32) bool {
	return GetRegistry().IsInitialized(ctx, characterId)
}

// InitializeCharacter performs lazy initialization of a character's effective
// stats. It fetches base stats + wearer profile from atlas-character, the
// equipped-asset snapshots from atlas-inventory, buffs from atlas-buffs,
// passive skill bonuses from atlas-data + atlas-skills, and finally runs the
// equipment qualification gate before publishing Computed to the registry.
func InitializeCharacter(l logrus.FieldLogger, ctx context.Context, characterId uint32, ch channel.Model) error {
	l.Debugf("Initializing effective stats for character [%d] on world [%d] channel [%d].", characterId, ch.WorldId(), ch.Id())

	m := GetRegistry().GetOrCreate(ctx, ch, characterId)
	if err := GetRegistry().MarkInitialized(ctx, characterId); err != nil {
		l.WithError(err).Warnf("Failed to mark character [%d] as initialized.", characterId)
	}

	baseStats, wp, err := fetchWearer(l, ctx, characterId)
	if err != nil {
		l.WithError(err).Warnf("Failed to fetch wearer record for character [%d], using defaults.", characterId)
		baseStats = stat.NewBase(0, 0, 0, 0, 0, 0)
		wp = WearerProfile{}
	}
	m = m.WithBaseStats(baseStats).WithWearer(wp)

	snapshots, err := fetchEquippedSnapshots(l, ctx, characterId)
	if err != nil {
		l.WithError(err).Warnf("Failed to fetch equipped snapshots for character [%d].", characterId)
	} else {
		for _, snap := range snapshots {
			m = m.WithEquippedAsset(snap)
		}
	}

	buffBonuses, err := fetchBuffBonuses(l, ctx, characterId)
	if err != nil {
		l.WithError(err).Warnf("Failed to fetch buff bonuses for character [%d].", characterId)
	} else {
		m = m.WithBonuses(buffBonuses)
	}

	passiveBonuses, err := fetchPassiveBonuses(l, ctx, characterId)
	if err != nil {
		l.WithError(err).Warnf("Failed to fetch passive skill bonuses for character [%d].", characterId)
	} else {
		m = m.WithBonuses(passiveBonuses)
	}

	m = m.RecomputeWith(equipment.GetProvider(l), ctx).WithInitialized()
	GetRegistry().Update(ctx, m)

	l.Debugf("Completed initialization for character [%d]. Effective stats: STR=%d, DEX=%d, INT=%d, LUK=%d, MaxHP=%d, MaxMP=%d",
		characterId, m.Computed().Strength(), m.Computed().Dexterity(), m.Computed().Intelligence(),
		m.Computed().Luck(), m.Computed().MaxHp(), m.Computed().MaxMp())

	return nil
}

// fetchWearer fetches the character record and returns base stats + wearer
// profile in a single call.
func fetchWearer(l logrus.FieldLogger, ctx context.Context, characterId uint32) (stat.Base, WearerProfile, error) {
	l.Debugf("Fetching base stats + wearer profile for character [%d] from character service.", characterId)

	charData, err := character.RequestById(characterId)(l, ctx)
	if err != nil {
		return stat.Base{}, WearerProfile{}, fmt.Errorf("failed to fetch character [%d]: %w", characterId, err)
	}

	base := stat.NewBase(
		charData.Strength,
		charData.Dexterity,
		charData.Luck,
		charData.Intelligence,
		charData.MaxHp,
		charData.MaxMp,
	)
	wp := NewWearerProfile(charData.Level, charData.JobId)
	return base, wp, nil
}

// fetchEquippedSnapshots iterates the equip compartment and returns a
// snapshot per equipped (Slot < 0) asset, with bonuses pre-extracted.
func fetchEquippedSnapshots(l logrus.FieldLogger, ctx context.Context, characterId uint32) ([]EquippedAsset, error) {
	l.Debugf("Fetching equipment snapshots for character [%d] from inventory service.", characterId)

	compartment, err := inventory.RequestEquipCompartment(characterId)(l, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch equip compartment for character [%d]: %w", characterId, err)
	}

	out := make([]EquippedAsset, 0)
	for _, asset := range compartment.Assets {
		if !asset.IsEquipped() {
			continue
		}
		equipData, ok := asset.GetEquipableData()
		if !ok {
			continue
		}
		bonuses := extractAssetBonuses(asset.Id, equipData)
		out = append(out, NewEquippedAsset(asset.Id, asset.TemplateId, bonuses))
	}

	l.Debugf("Built %d equipment snapshots for character [%d].", len(out), characterId)
	return out, nil
}

// extractAssetBonuses converts atlas-inventory equipable stats into the flat
// stat.Bonus list keyed by source = "equipment:<assetId>".
func extractAssetBonuses(assetId uint32, equipData inventory.EquipableRestData) []stat.Bonus {
	bonuses := make([]stat.Bonus, 0)
	source := fmt.Sprintf("equipment:%d", assetId)

	if equipData.Strength > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeStrength, int32(equipData.Strength)))
	}
	if equipData.Dexterity > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeDexterity, int32(equipData.Dexterity)))
	}
	if equipData.Luck > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeLuck, int32(equipData.Luck)))
	}
	if equipData.Intelligence > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeIntelligence, int32(equipData.Intelligence)))
	}
	if equipData.Hp > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeMaxHp, int32(equipData.Hp)))
	}
	if equipData.Mp > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeMaxMp, int32(equipData.Mp)))
	}
	if equipData.WeaponAttack > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeWeaponAttack, int32(equipData.WeaponAttack)))
	}
	if equipData.MagicAttack > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeMagicAttack, int32(equipData.MagicAttack)))
	}
	if equipData.WeaponDefense > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeWeaponDefense, int32(equipData.WeaponDefense)))
	}
	if equipData.MagicDefense > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeMagicDefense, int32(equipData.MagicDefense)))
	}
	if equipData.Accuracy > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeAccuracy, int32(equipData.Accuracy)))
	}
	if equipData.Avoidability > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeAvoidability, int32(equipData.Avoidability)))
	}
	if equipData.Speed > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeSpeed, int32(equipData.Speed)))
	}
	if equipData.Jump > 0 {
		bonuses = append(bonuses, stat.NewBonus(source, stat.TypeJump, int32(equipData.Jump)))
	}
	return bonuses
}

// fetchBuffBonuses fetches active buffs and their stat changes
func fetchBuffBonuses(l logrus.FieldLogger, ctx context.Context, characterId uint32) ([]stat.Bonus, error) {
	l.Debugf("Fetching buff bonuses for character [%d] from buffs service.", characterId)

	buffList, err := buffs.RequestCharacterBuffs(characterId)(l, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch buffs for character [%d]: %w", characterId, err)
	}

	bonuses := make([]stat.Bonus, 0)
	for _, buff := range buffList {
		source := fmt.Sprintf("buff:%d", buff.SourceId)
		for _, change := range buff.Changes {
			statType, isMultiplier := stat.MapBuffStatType(change.Type)
			if statType == "" {
				l.Debugf("Unknown buff stat type: %s", change.Type)
				continue
			}
			if isMultiplier {
				multiplier := float64(change.Amount) / 100.0
				bonuses = append(bonuses, stat.NewMultiplierBonus(source, statType, multiplier))
			} else {
				bonuses = append(bonuses, stat.NewBonus(source, statType, change.Amount))
			}
		}
	}

	l.Debugf("Fetched %d buff bonuses for character [%d].", len(bonuses), characterId)
	return bonuses, nil
}

// fetchPassiveBonuses fetches passive skill bonuses from character skills
func fetchPassiveBonuses(l logrus.FieldLogger, ctx context.Context, characterId uint32) ([]stat.Bonus, error) {
	l.Debugf("Fetching passive skill bonuses for character [%d].", characterId)

	characterSkills, err := skills.RequestCharacterSkills(characterId)(l, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch skills for character [%d]: %w", characterId, err)
	}

	bonuses := make([]stat.Bonus, 0)
	for _, charSkill := range characterSkills {
		if charSkill.Level == 0 {
			continue
		}
		skillInfo, err := skilldata.RequestById(charSkill.Id)(l, ctx)
		if err != nil {
			l.WithError(err).Debugf("Failed to fetch skill data for skill [%d], skipping.", charSkill.Id)
			continue
		}
		if !skillInfo.IsPassive() {
			continue
		}
		effect := skillInfo.GetEffectForLevel(charSkill.Level)
		if effect == nil {
			l.Debugf("No effect found for passive skill [%d] at level [%d].", charSkill.Id, charSkill.Level)
			continue
		}
		source := fmt.Sprintf("passive:%d", charSkill.Id)
		if effect.WeaponAttack != 0 {
			bonuses = append(bonuses, stat.NewBonus(source, stat.TypeWeaponAttack, int32(effect.WeaponAttack)))
		}
		if effect.MagicAttack != 0 {
			bonuses = append(bonuses, stat.NewBonus(source, stat.TypeMagicAttack, int32(effect.MagicAttack)))
		}
		if effect.WeaponDefense != 0 {
			bonuses = append(bonuses, stat.NewBonus(source, stat.TypeWeaponDefense, int32(effect.WeaponDefense)))
		}
		if effect.MagicDefense != 0 {
			bonuses = append(bonuses, stat.NewBonus(source, stat.TypeMagicDefense, int32(effect.MagicDefense)))
		}
		if effect.Accuracy != 0 {
			bonuses = append(bonuses, stat.NewBonus(source, stat.TypeAccuracy, int32(effect.Accuracy)))
		}
		if effect.Avoidability != 0 {
			bonuses = append(bonuses, stat.NewBonus(source, stat.TypeAvoidability, int32(effect.Avoidability)))
		}
		if effect.Speed != 0 {
			bonuses = append(bonuses, stat.NewBonus(source, stat.TypeSpeed, int32(effect.Speed)))
		}
		if effect.Jump != 0 {
			bonuses = append(bonuses, stat.NewBonus(source, stat.TypeJump, int32(effect.Jump)))
		}
		if effect.Hp > 0 {
			bonuses = append(bonuses, stat.NewBonus(source, stat.TypeMaxHp, int32(effect.Hp)))
		}
		if effect.Mp > 0 {
			bonuses = append(bonuses, stat.NewBonus(source, stat.TypeMaxMp, int32(effect.Mp)))
		}
		for _, statup := range effect.Statups {
			statType := stat.MapStatupType(statup.Type)
			if statType == "" {
				l.Debugf("Unknown passive stat type: %s for skill [%d].", statup.Type, charSkill.Id)
				continue
			}
			bonuses = append(bonuses, stat.NewBonus(source, statType, statup.Amount))
		}
	}

	l.Debugf("Fetched %d passive skill bonuses for character [%d].", len(bonuses), characterId)
	return bonuses, nil
}
