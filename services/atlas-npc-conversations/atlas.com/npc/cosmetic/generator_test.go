package cosmetic

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func testLogger() logrus.FieldLogger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Quiet during tests
	return logger
}

func testContext() context.Context {
	return context.Background()
}

func createTestCharacter(gender byte, hair uint32) CharacterAppearance {
	return NewCharacterAppearance(1, gender, hair, 20000, 0)
}

// Test hair style generation with color preservation
func TestGenerateHairStyles_PreserveColor(t *testing.T) {
	// Create a male character with hair style 3006, color 7 (white)
	char := createTestCharacter(0, 30067)
	baseStyles := []uint32{30060, 30140, 30200}

	gen := NewGenerator(testLogger(), testContext())
	result := gen.GenerateHairStyles(char, baseStyles, false, true)

	// All styles should have color 7 preserved
	expected := []uint32{30067, 30147, 30207}
	assert.Equal(t, expected, result)

	// Verify all have color 7
	for _, style := range result {
		assert.Equal(t, byte(7), byte(style%10), "Expected color 7 for style %d", style)
	}
}

// Test hair style generation with color normalization
func TestGenerateHairStyles_NormalizeColor(t *testing.T) {
	// Create character with hair color 5
	char := createTestCharacter(0, 30065)
	baseStyles := []uint32{30060, 30140, 30200}

	gen := NewGenerator(testLogger(), testContext())
	result := gen.GenerateHairStyles(char, baseStyles, false, false)

	// All styles should be normalized to color 0
	expected := []uint32{30060, 30140, 30200}
	assert.Equal(t, expected, result)

	// Verify all have color 0
	for _, style := range result {
		assert.Equal(t, byte(0), byte(style%10), "Expected color 0 for style %d", style)
	}
}

// Test gender filtering for male characters
func TestGenerateHairStyles_GenderFilter_Male(t *testing.T) {
	// Create male character
	char := createTestCharacter(0, 30060)
	// Mix of male (30060, 30140) and female (31150, 31300) styles
	baseStyles := []uint32{30060, 31150, 30140, 31300}

	gen := NewGenerator(testLogger(), testContext())
	result := gen.GenerateHairStyles(char, baseStyles, true, false)

	// Only male styles should remain
	expected := []uint32{30060, 30140}
	assert.Equal(t, expected, result)

	// Verify all are male styles (base 3000-3299)
	for _, style := range result {
		base := style / 10
		assert.True(t, base >= 3000 && base < 3300, "Expected male style, got %d", style)
	}
}

// Test gender filtering for female characters
func TestGenerateHairStyles_GenderFilter_Female(t *testing.T) {
	// Create female character
	char := createTestCharacter(1, 31150)
	// Mix of male and female styles
	baseStyles := []uint32{30060, 31150, 30140, 31300}

	gen := NewGenerator(testLogger(), testContext())
	result := gen.GenerateHairStyles(char, baseStyles, true, false)

	// Only female styles should remain
	expected := []uint32{31150, 31300}
	assert.Equal(t, expected, result)

	// Verify all are female styles (base 3100-3499)
	for _, style := range result {
		base := style / 10
		assert.True(t, base >= 3100 && base < 3500, "Expected female style, got %d", style)
	}
}

// Test gender filtering with color preservation
func TestGenerateHairStyles_GenderFilterAndPreserveColor(t *testing.T) {
	// Create male character with color 3 (blonde)
	char := createTestCharacter(0, 30063)
	baseStyles := []uint32{30060, 31150, 30140, 31300}

	gen := NewGenerator(testLogger(), testContext())
	result := gen.GenerateHairStyles(char, baseStyles, true, true)

	// Only male styles with color 3
	expected := []uint32{30063, 30143}
	assert.Equal(t, expected, result)
}

// Test hair color generation
func TestGenerateHairColors_AllColors(t *testing.T) {
	// Create character with style 3006
	char := createTestCharacter(0, 30060)
	colors := []byte{0, 1, 2, 3, 4, 5, 6, 7}

	gen := NewGenerator(testLogger(), testContext())
	result := gen.GenerateHairColors(char, colors)

	// Should generate all 8 color variants
	expected := []uint32{30060, 30061, 30062, 30063, 30064, 30065, 30066, 30067}
	assert.Equal(t, expected, result)
}

// Test hair color generation with subset of colors
func TestGenerateHairColors_SubsetColors(t *testing.T) {
	// Create character with style 3014
	char := createTestCharacter(0, 30140)
	colors := []byte{0, 2, 4, 6} // Even colors only

	gen := NewGenerator(testLogger(), testContext())
	result := gen.GenerateHairColors(char, colors)

	expected := []uint32{30140, 30142, 30144, 30146}
	assert.Equal(t, expected, result)
}

// Test face style generation with gender filter
func TestGenerateFaceStyles_GenderFilter_Male(t *testing.T) {
	// Create male character
	char := createTestCharacter(0, 30060)
	// Mix of male (20000-20999) and female (21000-21999) faces
	baseStyles := []uint32{20000, 21000, 20001, 21001}

	gen := NewGenerator(testLogger(), testContext())
	result := gen.GenerateFaceStyles(char, baseStyles, true)

	// Only male faces should remain
	expected := []uint32{20000, 20001}
	assert.Equal(t, expected, result)
}

// Test face style generation for female
func TestGenerateFaceStyles_GenderFilter_Female(t *testing.T) {
	// Create female character
	char := createTestCharacter(1, 31150)
	baseStyles := []uint32{20000, 21000, 20001, 21001}

	gen := NewGenerator(testLogger(), testContext())
	result := gen.GenerateFaceStyles(char, baseStyles, true)

	// Only female faces should remain
	expected := []uint32{21000, 21001}
	assert.Equal(t, expected, result)
}

// Test face style generation without gender filter
func TestGenerateFaceStyles_NoGenderFilter(t *testing.T) {
	char := createTestCharacter(0, 30060)
	baseStyles := []uint32{20000, 21000, 20001, 21001}

	gen := NewGenerator(testLogger(), testContext())
	result := gen.GenerateFaceStyles(char, baseStyles, false)

	// All faces should be returned
	assert.Equal(t, baseStyles, result)
}

// Test empty base styles
func TestGenerateHairStyles_EmptyBaseStyles(t *testing.T) {
	char := createTestCharacter(0, 30060)
	baseStyles := []uint32{}

	gen := NewGenerator(testLogger(), testContext())
	result := gen.GenerateHairStyles(char, baseStyles, false, false)

	assert.Empty(t, result)
}

// Test edge case: all styles filtered out by gender
func TestGenerateHairStyles_AllFilteredOut(t *testing.T) {
	// Male character
	char := createTestCharacter(0, 30060)
	// Only female styles
	baseStyles := []uint32{31150, 31300, 31740}

	gen := NewGenerator(testLogger(), testContext())
	result := gen.GenerateHairStyles(char, baseStyles, true, false)

	assert.Empty(t, result)
}

// Test CharacterAppearance helper methods
func TestCharacterAppearance_HairDecomposition(t *testing.T) {
	// Hair ID 30067 = base 3006 + color 7
	char := createTestCharacter(0, 30067)

	assert.Equal(t, uint32(3006), char.HairBase())
	assert.Equal(t, byte(7), char.HairColor())
}

func TestCharacterAppearance_GenderHelpers(t *testing.T) {
	maleChar := createTestCharacter(0, 30060)
	assert.True(t, maleChar.IsMale())
	assert.False(t, maleChar.IsFemale())

	femaleChar := createTestCharacter(1, 31150)
	assert.False(t, femaleChar.IsMale())
	assert.True(t, femaleChar.IsFemale())
}
