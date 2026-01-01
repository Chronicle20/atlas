package cosmetic

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor provides high-level cosmetic operations for NPC conversations
type Processor interface {
	GenerateHairStyles(characterId uint32, params map[string]string) ([]uint32, error)
	GenerateHairColors(characterId uint32, params map[string]string) ([]uint32, error)
	GenerateFaceStyles(characterId uint32, params map[string]string) ([]uint32, error)
	UpdateCharacterAppearance(characterId uint32, cosmeticType string, styleId uint32) error
}

type ProcessorImpl struct {
	l                   logrus.FieldLogger
	ctx                 context.Context
	generator           Generator
	validator           Validator
	appearanceProvider  AppearanceProvider
}

// AppearanceProvider is an interface for retrieving character appearance data
// This will be implemented by querying the atlas-query-aggregator service
type AppearanceProvider interface {
	GetCharacterAppearance(ctx context.Context, characterId uint32) (CharacterAppearance, error)
}

// NewProcessor creates a new Processor instance
func NewProcessor(l logrus.FieldLogger, ctx context.Context, appearanceProvider AppearanceProvider) Processor {
	return &ProcessorImpl{
		l:                  l,
		ctx:                ctx,
		generator:          NewGenerator(l, ctx),
		validator:          NewValidator(l, ctx),
		appearanceProvider: appearanceProvider,
	}
}

// GenerateHairStyles generates filtered hair style list based on character appearance and parameters
func (p *ProcessorImpl) GenerateHairStyles(
	characterId uint32,
	params map[string]string,
) ([]uint32, error) {
	p.l.Debugf("Generating hair styles for character %d with params: %v", characterId, params)

	// Get character appearance
	appearance, err := p.appearanceProvider.GetCharacterAppearance(p.ctx, characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to get character appearance for character %d", characterId)
		return nil, fmt.Errorf("failed to get character appearance: %w", err)
	}

	// Parse base styles from params
	baseStyles, err := p.parseUint32Array(params["baseStyles"])
	if err != nil {
		return nil, fmt.Errorf("invalid baseStyles parameter: %w", err)
	}

	if len(baseStyles) == 0 {
		return nil, fmt.Errorf("baseStyles parameter is empty")
	}

	// Parse boolean options
	genderFilter := params["genderFilter"] == "true"
	preserveColor := params["preserveColor"] == "true"
	validateExists := params["validateExists"] == "true"
	excludeEquipped := params["excludeEquipped"] == "true"

	p.l.Debugf("Options: genderFilter=%v, preserveColor=%v, validateExists=%v, excludeEquipped=%v",
		genderFilter, preserveColor, validateExists, excludeEquipped)

	// Generate styles
	styles := p.generator.GenerateHairStyles(appearance, baseStyles, genderFilter, preserveColor)

	// Validate if requested
	if validateExists {
		styles = p.validator.FilterValid(characterId, styles, CosmeticTypeHair, excludeEquipped)
	}

	if len(styles) == 0 {
		p.l.Warnf("No valid hair styles generated for character %d", characterId)
		return nil, fmt.Errorf("no valid hair styles available after filtering")
	}

	p.l.Infof("Generated %d hair styles for character %d", len(styles), characterId)
	return styles, nil
}

// GenerateHairColors generates color variants for the character's current hairstyle
func (p *ProcessorImpl) GenerateHairColors(
	characterId uint32,
	params map[string]string,
) ([]uint32, error) {
	p.l.Debugf("Generating hair colors for character %d with params: %v", characterId, params)

	// Get character appearance
	appearance, err := p.appearanceProvider.GetCharacterAppearance(p.ctx, characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to get character appearance for character %d", characterId)
		return nil, fmt.Errorf("failed to get character appearance: %w", err)
	}

	// Parse colors from params
	colorsStr := params["colors"]
	if colorsStr == "" {
		colorsStr = "0,1,2,3,4,5,6,7" // Default: all colors
	}

	colors, err := p.parseByteArray(colorsStr)
	if err != nil {
		return nil, fmt.Errorf("invalid colors parameter: %w", err)
	}

	// Parse boolean options
	validateExists := params["validateExists"] == "true"
	excludeEquipped := params["excludeEquipped"] == "true"

	p.l.Debugf("Options: validateExists=%v, excludeEquipped=%v", validateExists, excludeEquipped)

	// Generate color variants
	styles := p.generator.GenerateHairColors(appearance, colors)

	// Validate if requested
	if validateExists {
		styles = p.validator.FilterValid(characterId, styles, CosmeticTypeHair, excludeEquipped)
	}

	if len(styles) == 0 {
		p.l.Warnf("No valid hair color variants generated for character %d", characterId)
		return nil, fmt.Errorf("no valid hair color variants available after filtering")
	}

	p.l.Infof("Generated %d hair color variants for character %d", len(styles), characterId)
	return styles, nil
}

// GenerateFaceStyles generates filtered face style list based on character appearance and parameters
func (p *ProcessorImpl) GenerateFaceStyles(
	characterId uint32,
	params map[string]string,
) ([]uint32, error) {
	p.l.Debugf("Generating face styles for character %d with params: %v", characterId, params)

	// Get character appearance
	appearance, err := p.appearanceProvider.GetCharacterAppearance(p.ctx, characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to get character appearance for character %d", characterId)
		return nil, fmt.Errorf("failed to get character appearance: %w", err)
	}

	// Parse base styles from params
	baseStyles, err := p.parseUint32Array(params["baseStyles"])
	if err != nil {
		return nil, fmt.Errorf("invalid baseStyles parameter: %w", err)
	}

	if len(baseStyles) == 0 {
		return nil, fmt.Errorf("baseStyles parameter is empty")
	}

	// Parse boolean options
	genderFilter := params["genderFilter"] == "true"
	validateExists := params["validateExists"] == "true"
	excludeEquipped := params["excludeEquipped"] == "true"

	p.l.Debugf("Options: genderFilter=%v, validateExists=%v, excludeEquipped=%v",
		genderFilter, validateExists, excludeEquipped)

	// Generate styles
	styles := p.generator.GenerateFaceStyles(appearance, baseStyles, genderFilter)

	// Validate if requested
	if validateExists {
		styles = p.validator.FilterValid(characterId, styles, CosmeticTypeFace, excludeEquipped)
	}

	if len(styles) == 0 {
		p.l.Warnf("No valid face styles generated for character %d", characterId)
		return nil, fmt.Errorf("no valid face styles available after filtering")
	}

	p.l.Infof("Generated %d face styles for character %d", len(styles), characterId)
	return styles, nil
}

// parseUint32Array parses a comma-separated string into a uint32 array
func (p *ProcessorImpl) parseUint32Array(str string) ([]uint32, error) {
	if str == "" {
		return []uint32{}, nil
	}

	parts := strings.Split(str, ",")
	result := make([]uint32, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		val, err := strconv.ParseUint(trimmed, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid number '%s': %w", trimmed, err)
		}

		result = append(result, uint32(val))
	}

	return result, nil
}

// parseByteArray parses a comma-separated string into a byte array
func (p *ProcessorImpl) parseByteArray(str string) ([]byte, error) {
	if str == "" {
		return []byte{}, nil
	}

	parts := strings.Split(str, ",")
	result := make([]byte, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		val, err := strconv.ParseUint(trimmed, 10, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid byte value '%s': %w", trimmed, err)
		}

		result = append(result, byte(val))
	}

	return result, nil
}

// UpdateCharacterAppearance updates a character's cosmetic appearance via the character service
func (p *ProcessorImpl) UpdateCharacterAppearance(characterId uint32, cosmeticType string, styleId uint32) error {
	p.l.Infof("Updating %s to %d for character %d", cosmeticType, styleId, characterId)

	// Validate cosmetic type and create appropriate update request
	var updateRequest CharacterUpdateRequest
	var err error

	switch cosmeticType {
	case "hair":
		if styleId < 30000 || styleId > 35000 {
			return fmt.Errorf("invalid hair ID: %d (must be 30000-35000)", styleId)
		}
		updateRequest = NewHairUpdateRequest(styleId)

	case "face":
		if styleId < 20000 || styleId > 25000 {
			return fmt.Errorf("invalid face ID: %d (must be 20000-25000)", styleId)
		}
		updateRequest = NewFaceUpdateRequest(styleId)

	case "skin":
		if styleId > 9 {
			return fmt.Errorf("invalid skin color: %d (must be 0-9)", styleId)
		}
		updateRequest = NewSkinColorUpdateRequest(byte(styleId))

	default:
		return fmt.Errorf("invalid cosmeticType: %s (must be 'hair', 'face', or 'skin')", cosmeticType)
	}

	// Make PATCH request to character service
	provider := requests.Provider[RestCharacterModel, CharacterAppearance](p.l, p.ctx)(
		requestUpdateCharacter(characterId, updateRequest),
		ExtractAppearance,
	)

	_, err = provider()
	if err != nil {
		p.l.WithError(err).Errorf("Failed to update %s for character %d", cosmeticType, characterId)
		return fmt.Errorf("failed to update character appearance: %w", err)
	}

	p.l.Infof("Successfully updated %s to %d for character %d", cosmeticType, styleId, characterId)
	return nil
}
