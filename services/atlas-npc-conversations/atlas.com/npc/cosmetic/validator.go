package cosmetic

import (
	"context"

	"github.com/sirupsen/logrus"
)

// Validator checks style ownership (equipped status)
// Note: Style existence validation is handled by the processor via REST calls to atlas-data
type Validator interface {
	IsEquipped(characterId uint32, styleId uint32) bool
	FilterEquipped(characterId uint32, styles []uint32) []uint32
}

type ValidatorImpl struct {
	l                  logrus.FieldLogger
	ctx                context.Context
	appearanceProvider AppearanceProvider
}

// NewValidator creates a new Validator instance
func NewValidator(l logrus.FieldLogger, ctx context.Context) Validator {
	return &ValidatorImpl{l: l, ctx: ctx, appearanceProvider: nil}
}

// NewValidatorWithAppearance creates a new Validator instance with an appearance provider for equipped checks
func NewValidatorWithAppearance(l logrus.FieldLogger, ctx context.Context, ap AppearanceProvider) Validator {
	return &ValidatorImpl{l: l, ctx: ctx, appearanceProvider: ap}
}

// IsEquipped checks if a character currently has a style equipped
func (v *ValidatorImpl) IsEquipped(characterId uint32, styleId uint32) bool {
	if v.appearanceProvider == nil {
		// No appearance provider configured, can't check
		return false
	}

	appearance, err := v.appearanceProvider.GetCharacterAppearance(v.ctx, characterId)
	if err != nil {
		v.l.WithError(err).Warnf("Failed to get character %d appearance for equipped check", characterId)
		return false
	}

	// Check if the style matches current hair or face
	return appearance.Hair() == styleId || appearance.Face() == styleId
}

// FilterEquipped filters a list of styles, removing already-equipped styles
func (v *ValidatorImpl) FilterEquipped(characterId uint32, styles []uint32) []uint32 {
	result := make([]uint32, 0, len(styles))
	equippedCount := 0

	for _, styleId := range styles {
		if v.IsEquipped(characterId, styleId) {
			v.l.Debugf("Style %d is already equipped by character %d - excluding", styleId, characterId)
			equippedCount++
			continue
		}
		result = append(result, styleId)
	}

	if equippedCount > 0 {
		v.l.Infof("Filtered %d equipped styles for character %d, %d remaining",
			equippedCount, characterId, len(result))
	}

	return result
}
