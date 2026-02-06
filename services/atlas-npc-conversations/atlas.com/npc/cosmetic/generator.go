package cosmetic

import (
	"context"

	"github.com/sirupsen/logrus"
)

// Generator handles cosmetic style ID generation and filtering
type Generator interface {
	GenerateHairStyles(char CharacterAppearance, baseStyles []uint32, genderFilter bool, preserveColor bool) []uint32
	GenerateHairColors(char CharacterAppearance, colors []byte) []uint32
	GenerateFaceStyles(char CharacterAppearance, baseStyles []uint32, genderFilter bool) []uint32
	GenerateFaceColors(char CharacterAppearance, colorOffsets []uint32) []uint32
}

type GeneratorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewGenerator creates a new Generator instance
func NewGenerator(l logrus.FieldLogger, ctx context.Context) Generator {
	return &GeneratorImpl{l: l, ctx: ctx}
}

// GenerateHairStyles generates a list of hair style IDs based on character appearance and filters
func (g *GeneratorImpl) GenerateHairStyles(
	char CharacterAppearance,
	baseStyles []uint32,
	genderFilter bool,
	preserveColor bool,
) []uint32 {
	g.l.Debugf("Generating hair styles for character %d: baseStyles=%d, genderFilter=%v, preserveColor=%v",
		char.CharacterId(), len(baseStyles), genderFilter, preserveColor)

	styles := baseStyles

	// Filter by gender if requested
	if genderFilter {
		styles = g.filterByGender(styles, char.Gender())
		g.l.Debugf("After gender filter: %d styles remain", len(styles))
	}

	// Apply color transformation
	if preserveColor {
		currentColor := char.HairColor()
		styles = g.applyColorToStyles(styles, currentColor)
		g.l.Debugf("Applied color %d to %d styles", currentColor, len(styles))
	} else {
		styles = g.normalizeToBaseColor(styles)
		g.l.Debugf("Normalized %d styles to base color", len(styles))
	}

	return styles
}

// GenerateHairColors generates color variants for the character's current hairstyle
func (g *GeneratorImpl) GenerateHairColors(
	char CharacterAppearance,
	colors []byte,
) []uint32 {
	baseStyle := char.HairBase() * 10
	result := make([]uint32, 0, len(colors))

	g.l.Debugf("Generating hair colors for character %d: base=%d, colors=%d",
		char.CharacterId(), char.HairBase(), len(colors))

	for _, color := range colors {
		styleId := baseStyle + uint32(color)
		result = append(result, styleId)
	}

	g.l.Debugf("Generated %d color variants", len(result))
	return result
}

// GenerateFaceStyles generates a list of face style IDs based on character appearance and filters
func (g *GeneratorImpl) GenerateFaceStyles(
	char CharacterAppearance,
	baseStyles []uint32,
	genderFilter bool,
) []uint32 {
	g.l.Debugf("Generating face styles for character %d: baseStyles=%d, genderFilter=%v",
		char.CharacterId(), len(baseStyles), genderFilter)

	styles := baseStyles

	// Filter by gender if requested
	if genderFilter {
		styles = g.filterByGenderFace(styles, char.Gender())
		g.l.Debugf("After gender filter: %d face styles remain", len(styles))
	}

	return styles
}

// GenerateFaceColors generates color variants for the character's current face
// Face colors work differently than hair colors:
//   - Hair: base * 10 + color (e.g., 30067 = base 3006, color 7)
//   - Face: genderOffset + baseStyle + (colorIndex * 100)
//     e.g., 20401 = male (20000) + base 1 + color 4 (400)
//
// The colorOffsets parameter specifies which color offsets to generate
// Common values: 0 (base), 100 (color 1), 200, 300, 400, 500, 600, 700
func (g *GeneratorImpl) GenerateFaceColors(
	char CharacterAppearance,
	colorOffsets []uint32,
) []uint32 {
	// Get the base face style (0-99) from current face
	baseFace := char.FaceBase()

	// Determine gender offset (20000 for male, 21000 for female)
	var genderOffset uint32 = 20000
	if char.IsFemale() {
		genderOffset = 21000
	}

	// Calculate base face ID (gender offset + base style)
	baseFaceId := genderOffset + baseFace

	result := make([]uint32, 0, len(colorOffsets))

	g.l.Debugf("Generating face colors for character %d: currentFace=%d, baseFace=%d, genderOffset=%d, colorOffsets=%d",
		char.CharacterId(), char.Face(), baseFace, genderOffset, len(colorOffsets))

	for _, offset := range colorOffsets {
		faceId := baseFaceId + offset
		result = append(result, faceId)
	}

	g.l.Debugf("Generated %d face color variants", len(result))
	return result
}

// filterByGender filters hair styles based on gender
// Male styles: base 3000-3099 (30000-30999)
// Female styles: base 3100+ (31000+)
func (g *GeneratorImpl) filterByGender(styles []uint32, gender byte) []uint32 {
	result := make([]uint32, 0, len(styles))

	for _, styleId := range styles {
		base := g.getBaseStyle(styleId)

		if gender == 0 { // Male
			// Male hair: base 3000-3099
			if base >= 3000 && base < 3100 {
				result = append(result, styleId)
			}
		} else if gender == 1 { // Female
			// Female hair: base 3100+
			if base >= 3100 {
				result = append(result, styleId)
			}
		}
	}

	return result
}

// filterByGenderFace filters face styles based on gender
// Male faces: 20000-20999
// Female faces: 21000-21999
func (g *GeneratorImpl) filterByGenderFace(styles []uint32, gender byte) []uint32 {
	result := make([]uint32, 0, len(styles))

	for _, styleId := range styles {
		if gender == 0 { // Male
			if styleId >= 20000 && styleId < 21000 {
				result = append(result, styleId)
			}
		} else if gender == 1 { // Female
			if styleId >= 21000 && styleId < 22000 {
				result = append(result, styleId)
			}
		}
	}

	return result
}

// applyColorToStyles applies a specific color to a list of base styles
func (g *GeneratorImpl) applyColorToStyles(styles []uint32, color byte) []uint32 {
	result := make([]uint32, 0, len(styles))

	for _, styleId := range styles {
		coloredStyle := g.getColorVariant(styleId, color)
		result = append(result, coloredStyle)
	}

	return result
}

// normalizeToBaseColor normalizes all styles to color 0 (base color)
func (g *GeneratorImpl) normalizeToBaseColor(styles []uint32) []uint32 {
	result := make([]uint32, 0, len(styles))

	for _, styleId := range styles {
		baseStyle := g.getBaseStyle(styleId)
		normalizedStyle := baseStyle * 10 // Color 0
		result = append(result, normalizedStyle)
	}

	return result
}

// getBaseStyle extracts the base style from a full style ID
// Example: 30067 -> 3006
func (g *GeneratorImpl) getBaseStyle(styleId uint32) uint32 {
	return styleId / 10
}

// getColorVariant creates a style ID with a specific color
// Example: base=3006, color=7 -> 30067
func (g *GeneratorImpl) getColorVariant(styleId uint32, color byte) uint32 {
	base := g.getBaseStyle(styleId)
	return base*10 + uint32(color)
}
