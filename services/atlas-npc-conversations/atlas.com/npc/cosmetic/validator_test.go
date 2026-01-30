package cosmetic

import (
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// MockAppearanceProvider implements AppearanceProvider for testing
type MockAppearanceProvider struct {
	appearance CharacterAppearance
	err        error
}

func (m *MockAppearanceProvider) GetCharacterAppearance(ctx context.Context, characterId uint32) (CharacterAppearance, error) {
	if m.err != nil {
		return CharacterAppearance{}, m.err
	}
	return m.appearance, nil
}

func testValidatorLogger() logrus.FieldLogger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Quiet during tests
	return logger
}

func testValidatorContext() context.Context {
	return context.Background()
}

// Test IsEquipped returns true when style matches hair
func TestValidator_IsEquipped_MatchesHair(t *testing.T) {
	appearance := NewCharacterAppearance(1, 0, 30067, 20000, 0)
	mockProvider := &MockAppearanceProvider{appearance: appearance}

	v := NewValidatorWithAppearance(testValidatorLogger(), testValidatorContext(), mockProvider)

	// Should return true when styleId matches hair
	result := v.IsEquipped(1, 30067)
	assert.True(t, result)
}

// Test IsEquipped returns true when style matches face
func TestValidator_IsEquipped_MatchesFace(t *testing.T) {
	appearance := NewCharacterAppearance(1, 0, 30067, 20100, 0)
	mockProvider := &MockAppearanceProvider{appearance: appearance}

	v := NewValidatorWithAppearance(testValidatorLogger(), testValidatorContext(), mockProvider)

	// Should return true when styleId matches face
	result := v.IsEquipped(1, 20100)
	assert.True(t, result)
}

// Test IsEquipped returns false when style doesn't match
func TestValidator_IsEquipped_NoMatch(t *testing.T) {
	appearance := NewCharacterAppearance(1, 0, 30067, 20100, 0)
	mockProvider := &MockAppearanceProvider{appearance: appearance}

	v := NewValidatorWithAppearance(testValidatorLogger(), testValidatorContext(), mockProvider)

	// Should return false when styleId doesn't match hair or face
	result := v.IsEquipped(1, 30000)
	assert.False(t, result)
}

// Test IsEquipped returns false when appearance provider is nil
func TestValidator_IsEquipped_NilProvider(t *testing.T) {
	v := NewValidator(testValidatorLogger(), testValidatorContext())

	// Should return false when no appearance provider configured
	result := v.IsEquipped(1, 30067)
	assert.False(t, result)
}

// Test IsEquipped returns false when appearance provider returns error
func TestValidator_IsEquipped_ProviderError(t *testing.T) {
	mockProvider := &MockAppearanceProvider{err: errors.New("service unavailable")}

	v := NewValidatorWithAppearance(testValidatorLogger(), testValidatorContext(), mockProvider)

	// Should return false when provider fails
	result := v.IsEquipped(1, 30067)
	assert.False(t, result)
}

// Test FilterEquipped removes equipped styles
func TestValidator_FilterEquipped_RemovesEquipped(t *testing.T) {
	appearance := NewCharacterAppearance(1, 0, 30067, 20100, 0)
	mockProvider := &MockAppearanceProvider{appearance: appearance}

	v := NewValidatorWithAppearance(testValidatorLogger(), testValidatorContext(), mockProvider)

	styles := []uint32{30060, 30067, 30070, 20100, 20200}
	result := v.FilterEquipped(1, styles)

	// Should filter out 30067 (equipped hair) and 20100 (equipped face)
	expected := []uint32{30060, 30070, 20200}
	assert.Equal(t, expected, result)
}

// Test FilterEquipped returns all styles when none equipped
func TestValidator_FilterEquipped_NoneEquipped(t *testing.T) {
	appearance := NewCharacterAppearance(1, 0, 30067, 20100, 0)
	mockProvider := &MockAppearanceProvider{appearance: appearance}

	v := NewValidatorWithAppearance(testValidatorLogger(), testValidatorContext(), mockProvider)

	styles := []uint32{30060, 30070, 20200, 20300}
	result := v.FilterEquipped(1, styles)

	// All styles should remain since none match equipped
	assert.Equal(t, styles, result)
}

// Test FilterEquipped with empty input
func TestValidator_FilterEquipped_EmptyInput(t *testing.T) {
	appearance := NewCharacterAppearance(1, 0, 30067, 20100, 0)
	mockProvider := &MockAppearanceProvider{appearance: appearance}

	v := NewValidatorWithAppearance(testValidatorLogger(), testValidatorContext(), mockProvider)

	styles := []uint32{}
	result := v.FilterEquipped(1, styles)

	assert.Empty(t, result)
}

// Test FilterEquipped when all styles are equipped
func TestValidator_FilterEquipped_AllEquipped(t *testing.T) {
	appearance := NewCharacterAppearance(1, 0, 30067, 20100, 0)
	mockProvider := &MockAppearanceProvider{appearance: appearance}

	v := NewValidatorWithAppearance(testValidatorLogger(), testValidatorContext(), mockProvider)

	styles := []uint32{30067, 20100}
	result := v.FilterEquipped(1, styles)

	assert.Empty(t, result)
}

// Test FilterEquipped with nil provider returns all styles
func TestValidator_FilterEquipped_NilProvider(t *testing.T) {
	v := NewValidator(testValidatorLogger(), testValidatorContext())

	styles := []uint32{30060, 30067, 30070}
	result := v.FilterEquipped(1, styles)

	// With no provider, IsEquipped returns false, so all styles remain
	assert.Equal(t, styles, result)
}

// Test FilterEquipped with provider error returns all styles
func TestValidator_FilterEquipped_ProviderError(t *testing.T) {
	mockProvider := &MockAppearanceProvider{err: errors.New("service unavailable")}

	v := NewValidatorWithAppearance(testValidatorLogger(), testValidatorContext(), mockProvider)

	styles := []uint32{30060, 30067, 30070}
	result := v.FilterEquipped(1, styles)

	// With provider error, IsEquipped returns false, so all styles remain
	assert.Equal(t, styles, result)
}
