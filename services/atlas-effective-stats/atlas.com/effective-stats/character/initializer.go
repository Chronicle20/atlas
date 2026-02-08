package character

import (
	"atlas-effective-stats/external/buffs"
	"atlas-effective-stats/external/character"
	skilldata "atlas-effective-stats/external/data/skill"
	"atlas-effective-stats/external/inventory"
	"atlas-effective-stats/external/skills"
	"atlas-effective-stats/stat"
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

// IsInitialized checks if a character has been initialized
func IsInitialized(t tenant.Model, characterId uint32) bool {
	return GetRegistry().IsInitialized(t, characterId)
}

// InitializeCharacter performs lazy initialization of a character's effective stats
// This is called when stats are first requested for a character that hasn't been initialized
func InitializeCharacter(l logrus.FieldLogger, ctx context.Context, characterId uint32, ch channel.Model) error {
	t := tenant.MustFromContext(ctx)

	l.Debugf("Initializing effective stats for character [%d] on world [%d] channel [%d].", characterId, ch.WorldId(), ch.Id())

	// Create or get existing model
	m := GetRegistry().GetOrCreate(t, ch, characterId)

	// Mark as initialized first to prevent recursive initialization
	if err := GetRegistry().MarkInitialized(t, characterId); err != nil {
		l.WithError(err).Warnf("Failed to mark character [%d] as initialized.", characterId)
	}

	// Fetch base stats from character service
	baseStats, err := fetchBaseStats(l, ctx, characterId)
	if err != nil {
		l.WithError(err).Warnf("Failed to fetch base stats for character [%d], using defaults.", characterId)
		baseStats = stat.NewBase(0, 0, 0, 0, 0, 0)
	}
	m = m.WithBaseStats(baseStats)

	// Fetch equipped items from inventory service and their bonuses
	equipmentBonuses, err := fetchEquipmentBonuses(l, ctx, characterId)
	if err != nil {
		l.WithError(err).Warnf("Failed to fetch equipment bonuses for character [%d].", characterId)
	} else {
		m = m.WithBonuses(equipmentBonuses)
	}

	// Fetch active buffs from buffs service and their stat changes
	buffBonuses, err := fetchBuffBonuses(l, ctx, characterId)
	if err != nil {
		l.WithError(err).Warnf("Failed to fetch buff bonuses for character [%d].", characterId)
	} else {
		m = m.WithBonuses(buffBonuses)
	}

	// Fetch passive skill bonuses from character skills + data service
	passiveBonuses, err := fetchPassiveBonuses(l, ctx, characterId)
	if err != nil {
		l.WithError(err).Warnf("Failed to fetch passive skill bonuses for character [%d].", characterId)
	} else {
		m = m.WithBonuses(passiveBonuses)
	}

	// Compute final effective stats and update registry
	m = m.Recompute().WithInitialized()
	GetRegistry().Update(t, m)

	l.Debugf("Completed initialization for character [%d]. Effective stats: STR=%d, DEX=%d, INT=%d, LUK=%d, MaxHP=%d, MaxMP=%d",
		characterId, m.Computed().Strength(), m.Computed().Dexterity(), m.Computed().Intelligence(),
		m.Computed().Luck(), m.Computed().MaxHp(), m.Computed().MaxMp())

	return nil
}

// fetchBaseStats fetches base stats from the character service
func fetchBaseStats(l logrus.FieldLogger, ctx context.Context, characterId uint32) (stat.Base, error) {
	l.Debugf("Fetching base stats for character [%d] from character service.", characterId)

	charData, err := character.RequestById(characterId)(l, ctx)
	if err != nil {
		return stat.Base{}, fmt.Errorf("failed to fetch character [%d]: %w", characterId, err)
	}

	return stat.NewBase(
		charData.Strength,
		charData.Dexterity,
		charData.Luck,
		charData.Intelligence,
		charData.MaxHp,
		charData.MaxMp,
	), nil
}

// fetchEquipmentBonuses fetches equipped items and their stat bonuses
func fetchEquipmentBonuses(l logrus.FieldLogger, ctx context.Context, characterId uint32) ([]stat.Bonus, error) {
	l.Debugf("Fetching equipment bonuses for character [%d] from inventory service.", characterId)

	compartment, err := inventory.RequestEquipCompartment(characterId)(l, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch equip compartment for character [%d]: %w", characterId, err)
	}

	bonuses := make([]stat.Bonus, 0)

	for _, asset := range compartment.Assets {
		// Only process equipped items (negative slot positions)
		if !asset.IsEquipped() {
			continue
		}

		equipData, ok := asset.GetEquipableData()
		if !ok {
			continue
		}

		source := fmt.Sprintf("equipment:%d", asset.Id)

		// Add all equipment stats as bonuses
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
	}

	l.Debugf("Fetched %d equipment bonuses for character [%d].", len(bonuses), characterId)
	return bonuses, nil
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
				// Percentage buff (e.g., HYPER_BODY_HP gives 60% = 0.60 multiplier)
				multiplier := float64(change.Amount) / 100.0
				bonuses = append(bonuses, stat.NewMultiplierBonus(source, statType, multiplier))
			} else {
				// Flat buff
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

	// Fetch all character skills from atlas-skills service
	characterSkills, err := skills.RequestCharacterSkills(characterId)(l, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch skills for character [%d]: %w", characterId, err)
	}

	bonuses := make([]stat.Bonus, 0)

	// For each skill, check if it's a passive skill and extract bonuses
	for _, charSkill := range characterSkills {
		if charSkill.Level == 0 {
			continue
		}

		// Fetch skill data from atlas-data service
		skillInfo, err := skilldata.RequestById(charSkill.Id)(l, ctx)
		if err != nil {
			l.WithError(err).Debugf("Failed to fetch skill data for skill [%d], skipping.", charSkill.Id)
			continue
		}

		// Only process passive skills (action = false)
		if !skillInfo.IsPassive() {
			continue
		}

		// Get the effect for the character's skill level
		effect := skillInfo.GetEffectForLevel(charSkill.Level)
		if effect == nil {
			l.Debugf("No effect found for passive skill [%d] at level [%d].", charSkill.Id, charSkill.Level)
			continue
		}

		source := fmt.Sprintf("passive:%d", charSkill.Id)

		// Extract stat bonuses from the effect's direct stat fields
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

		// Extract bonuses from statups array
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
