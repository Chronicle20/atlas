package cosmetic

import (
	"context"
	"github.com/sirupsen/logrus"
)

// Validator checks style availability and ownership
type Validator interface {
	StyleExists(styleId uint32, styleType CosmeticType) bool
	IsEquipped(characterId uint32, styleId uint32) bool
	FilterValid(characterId uint32, styles []uint32, styleType CosmeticType, excludeEquipped bool) []uint32
}

type ValidatorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewValidator creates a new Validator instance
func NewValidator(l logrus.FieldLogger, ctx context.Context) Validator {
	return &ValidatorImpl{l: l, ctx: ctx}
}

// StyleExists checks if a style ID exists and is valid
// TODO Phase 2: Integrate with WZ data registry for actual validation
func (v *ValidatorImpl) StyleExists(styleId uint32, styleType CosmeticType) bool {
	// Basic range validation for now
	switch styleType {
	case CosmeticTypeHair:
		// Hair IDs: 30000-49999
		return styleId >= 30000 && styleId <= 49999
	case CosmeticTypeFace:
		// Face IDs: 20000-29999
		return styleId >= 20000 && styleId <= 29999
	case CosmeticTypeSkin:
		// Skin colors: 0-13
		return styleId >= 0 && styleId <= 13
	}
	return false
}

// IsEquipped checks if a character currently has a style equipped
// TODO Phase 2: Integrate with character equipment query
func (v *ValidatorImpl) IsEquipped(characterId uint32, styleId uint32) bool {
	// Placeholder implementation - allow all styles for now
	return false
}

// FilterValid filters a list of styles, removing invalid or already-equipped styles
func (v *ValidatorImpl) FilterValid(
	characterId uint32,
	styles []uint32,
	styleType CosmeticType,
	excludeEquipped bool,
) []uint32 {
	result := make([]uint32, 0, len(styles))
	invalidCount := 0
	equippedCount := 0

	for _, styleId := range styles {
		// Check existence
		if !v.StyleExists(styleId, styleType) {
			v.l.Warnf("Style %d (type %s) does not exist - excluding from list", styleId, styleType)
			invalidCount++
			continue
		}

		// Check if equipped
		if excludeEquipped && v.IsEquipped(characterId, styleId) {
			v.l.Debugf("Style %d is already equipped by character %d - excluding", styleId, characterId)
			equippedCount++
			continue
		}

		result = append(result, styleId)
	}

	if invalidCount > 0 || equippedCount > 0 {
		v.l.Infof("Filtered styles for character %d: %d invalid, %d equipped, %d remaining",
			characterId, invalidCount, equippedCount, len(result))
	}

	return result
}
