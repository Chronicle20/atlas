package cosmetic

import (
	"atlas-npc-conversations/cosmetic/data"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// InventoryChecker is an interface for checking item inventory
type InventoryChecker interface {
	HasItem(characterId uint32, itemId uint32) (bool, error)
}

// Processor provides high-level cosmetic operations for NPC conversations
type Processor interface {
	GenerateHairStyles(characterId uint32, params map[string]string) ([]uint32, error)
	GenerateHairColors(characterId uint32, params map[string]string) ([]uint32, error)
	GenerateFaceStyles(characterId uint32, params map[string]string) ([]uint32, error)
	GenerateFaceColors(characterId uint32, params map[string]string) ([]uint32, error)
	GenerateFaceColorsForOnetimeLens(characterId uint32, inventoryChecker InventoryChecker, params map[string]string) ([]uint32, error)
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
		validator:          NewValidatorWithAppearance(l, ctx, appearanceProvider),
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

	// Filter by existence if requested (calls atlas-data REST)
	if validateExists {
		styles = p.filterByExistence(styles, CosmeticTypeHair)
	}

	// Filter out equipped styles if requested
	if excludeEquipped {
		styles = p.validator.FilterEquipped(characterId, styles)
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

	// Filter by existence if requested (calls atlas-data REST)
	if validateExists {
		styles = p.filterByExistence(styles, CosmeticTypeHair)
	}

	// Filter out equipped styles if requested
	if excludeEquipped {
		styles = p.validator.FilterEquipped(characterId, styles)
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

	// Filter by existence if requested (calls atlas-data REST)
	if validateExists {
		styles = p.filterByExistence(styles, CosmeticTypeFace)
	}

	// Filter out equipped styles if requested
	if excludeEquipped {
		styles = p.validator.FilterEquipped(characterId, styles)
	}

	if len(styles) == 0 {
		p.l.Warnf("No valid face styles generated for character %d", characterId)
		return nil, fmt.Errorf("no valid face styles available after filtering")
	}

	p.l.Infof("Generated %d face styles for character %d", len(styles), characterId)
	return styles, nil
}

// GenerateFaceColors generates color variants for the character's current face
// This is used for cosmetic lens NPCs that change eye color
func (p *ProcessorImpl) GenerateFaceColors(
	characterId uint32,
	params map[string]string,
) ([]uint32, error) {
	p.l.Debugf("Generating face colors for character %d with params: %v", characterId, params)

	// Get character appearance
	appearance, err := p.appearanceProvider.GetCharacterAppearance(p.ctx, characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to get character appearance for character %d", characterId)
		return nil, fmt.Errorf("failed to get character appearance: %w", err)
	}

	// Parse color offsets from params
	// Colors are specified as offsets: 0, 100, 200, 300, 400, 500, 600, 700
	// These correspond to eye color indices 0-7
	colorOffsetsStr := params["colorOffsets"]
	if colorOffsetsStr == "" {
		colorOffsetsStr = "100,300,400,700" // Default: common lens colors (indices 1, 3, 4, 7)
	}

	colorOffsets, err := p.parseUint32Array(colorOffsetsStr)
	if err != nil {
		return nil, fmt.Errorf("invalid colorOffsets parameter: %w", err)
	}

	// Parse boolean options
	validateExists := params["validateExists"] == "true"
	excludeEquipped := params["excludeEquipped"] == "true"

	p.l.Debugf("Options: validateExists=%v, excludeEquipped=%v, colorOffsets=%v",
		validateExists, excludeEquipped, colorOffsets)

	// Generate color variants
	styles := p.generator.GenerateFaceColors(appearance, colorOffsets)

	// Filter by existence if requested (calls atlas-data REST)
	if validateExists {
		styles = p.filterByExistence(styles, CosmeticTypeFace)
	}

	// Filter out equipped styles if requested
	if excludeEquipped {
		styles = p.validator.FilterEquipped(characterId, styles)
	}

	if len(styles) == 0 {
		p.l.Warnf("No valid face color variants generated for character %d", characterId)
		return nil, fmt.Errorf("no valid face color variants available after filtering")
	}

	p.l.Infof("Generated %d face color variants for character %d", len(styles), characterId)
	return styles, nil
}

// GenerateFaceColorsForOnetimeLens generates face colors based on which one-time lens items the character owns
// One-time lens items are 5152100-5152107, mapping to face color offsets 0-700 (in steps of 100)
// This is used for NPCs like Dr.Roberts that allow using one-time cosmetic lens coupons
func (p *ProcessorImpl) GenerateFaceColorsForOnetimeLens(
	characterId uint32,
	inventoryChecker InventoryChecker,
	params map[string]string,
) ([]uint32, error) {
	p.l.Debugf("Generating face colors for one-time lens items for character %d with params: %v", characterId, params)

	// Get character appearance
	appearance, err := p.appearanceProvider.GetCharacterAppearance(p.ctx, characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to get character appearance for character %d", characterId)
		return nil, fmt.Errorf("failed to get character appearance: %w", err)
	}

	// Check which one-time lens items the character has (5152100-5152107)
	// Each item maps to a face color offset: item 5152100 -> offset 0, item 5152101 -> offset 100, etc.
	validColorOffsets := make([]uint32, 0, 8)
	for i := uint32(0); i < 8; i++ {
		itemId := 5152100 + i
		hasItem, err := inventoryChecker.HasItem(characterId, itemId)
		if err != nil {
			p.l.WithError(err).Warnf("Failed to check item %d for character %d, skipping", itemId, characterId)
			continue
		}
		if hasItem {
			colorOffset := i * 100
			validColorOffsets = append(validColorOffsets, colorOffset)
			p.l.Debugf("Character %d has item %d, adding color offset %d", characterId, itemId, colorOffset)
		}
	}

	if len(validColorOffsets) == 0 {
		p.l.Warnf("Character %d has no one-time lens items", characterId)
		return nil, fmt.Errorf("no one-time lens items found")
	}

	// Parse boolean options
	validateExists := params["validateExists"] == "true"
	excludeEquipped := params["excludeEquipped"] == "true"

	p.l.Debugf("Options: validateExists=%v, excludeEquipped=%v, validColorOffsets=%v",
		validateExists, excludeEquipped, validColorOffsets)

	// Generate color variants for the offsets where the player has items
	styles := p.generator.GenerateFaceColors(appearance, validColorOffsets)

	// Filter by existence if requested (calls atlas-data REST)
	if validateExists {
		styles = p.filterByExistence(styles, CosmeticTypeFace)
	}

	// Filter out equipped styles if requested
	if excludeEquipped {
		styles = p.validator.FilterEquipped(characterId, styles)
	}

	if len(styles) == 0 {
		p.l.Warnf("No valid face color variants generated for character %d after filtering", characterId)
		return nil, fmt.Errorf("no valid face color variants available after filtering")
	}

	p.l.Infof("Generated %d face color variants for one-time lens for character %d", len(styles), characterId)
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

// hairExists checks if a hair style exists in atlas-data via REST call
func (p *ProcessorImpl) hairExists(hairId uint32) bool {
	// Quick range check to avoid unnecessary API calls
	if hairId < 30000 || hairId > 49999 {
		return false
	}

	// Query atlas-data service - 404 means not found
	_, err := data.RequestHairById(hairId)(p.l, p.ctx)
	if err != nil {
		p.l.Debugf("Hair %d not found in atlas-data (may not exist or service unavailable)", hairId)
		return false
	}
	return true
}

// faceExists checks if a face style exists in atlas-data via REST call
func (p *ProcessorImpl) faceExists(faceId uint32) bool {
	// Quick range check to avoid unnecessary API calls
	if faceId < 20000 || faceId > 29999 {
		return false
	}

	// Query atlas-data service - 404 means not found
	_, err := data.RequestFaceById(faceId)(p.l, p.ctx)
	if err != nil {
		p.l.Debugf("Face %d not found in atlas-data (may not exist or service unavailable)", faceId)
		return false
	}
	return true
}

// skinExists checks if a skin color is valid (hardcoded, not in WZ data)
func (p *ProcessorImpl) skinExists(skinId uint32) bool {
	return skinId <= 13
}

// filterByExistence filters styles by checking existence via atlas-data REST calls
func (p *ProcessorImpl) filterByExistence(styles []uint32, styleType CosmeticType) []uint32 {
	result := make([]uint32, 0, len(styles))
	invalidCount := 0

	for _, styleId := range styles {
		var exists bool
		switch styleType {
		case CosmeticTypeHair:
			exists = p.hairExists(styleId)
		case CosmeticTypeFace:
			exists = p.faceExists(styleId)
		case CosmeticTypeSkin:
			exists = p.skinExists(styleId)
		}

		if !exists {
			p.l.Warnf("Style %d (type %s) does not exist - excluding from list", styleId, styleType)
			invalidCount++
			continue
		}
		result = append(result, styleId)
	}

	if invalidCount > 0 {
		p.l.Infof("Filtered %d non-existent styles, %d remaining", invalidCount, len(result))
	}
	return result
}
