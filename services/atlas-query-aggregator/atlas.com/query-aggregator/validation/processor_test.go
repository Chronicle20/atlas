package validation

import (
	"atlas-query-aggregator/character"
	"atlas-query-aggregator/character/mock"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"strings"
	"testing"
)

// TestProcessorValidate tests the Validate function of the validation processor
func TestProcessorValidate(t *testing.T) {
	// Create a logger
	logger := logrus.New()

	// Test cases
	tests := []struct {
		name               string
		characterId        uint32
		conditions         []string
		decorators         []model.Decorator[ValidationResult]
		setupMock          func(*mock.ProcessorImpl)
		wantPassed         bool
		wantDetailsCount   int
		wantError          bool
		wantErrorContains  string
	}{
		{
			name:        "All conditions pass",
			characterId: 123,
			conditions:  []string{"jobId=100", "meso>=10000", "mapId=2000", "fame>=50"},
			setupMock: func(m *mock.ProcessorImpl) {
				m.GetByIdFunc = func(decorators ...model.Decorator[character.Model]) func(characterId uint32) (character.Model, error) {
					return func(characterId uint32) (character.Model, error) {
						return character.NewModelBuilder().
							SetJobId(100).
							SetMeso(10000).
							SetMapId(2000).
							SetFame(50).
							Build(), nil
					}
				}
			},
			wantPassed:       true,
			wantDetailsCount: 4,
			wantError:        false,
		},
		{
			name:        "Some conditions fail",
			characterId: 123,
			conditions:  []string{"jobId=100", "meso>=20000", "mapId=2000", "fame>=60"},
			setupMock: func(m *mock.ProcessorImpl) {
				m.GetByIdFunc = func(decorators ...model.Decorator[character.Model]) func(characterId uint32) (character.Model, error) {
					return func(characterId uint32) (character.Model, error) {
						return character.NewModelBuilder().
							SetJobId(100).
							SetMeso(10000).
							SetMapId(2000).
							SetFame(50).
							Build(), nil
					}
				}
			},
			wantPassed:       false,
			wantDetailsCount: 4,
			wantError:        false,
		},
		{
			name:        "Error getting character data",
			characterId: 123,
			conditions:  []string{"jobId=100"},
			setupMock: func(m *mock.ProcessorImpl) {
				m.GetByIdFunc = func(decorators ...model.Decorator[character.Model]) func(characterId uint32) (character.Model, error) {
					return func(characterId uint32) (character.Model, error) {
						return character.Model{}, errors.New("character not found")
					}
				}
			},
			wantPassed:        false,
			wantDetailsCount:  0,
			wantError:         true,
			wantErrorContains: "failed to get character data",
		},
		{
			name:        "Invalid condition",
			characterId: 123,
			conditions:  []string{"invalid=100"},
			setupMock: func(m *mock.ProcessorImpl) {
				m.GetByIdFunc = func(decorators ...model.Decorator[character.Model]) func(characterId uint32) (character.Model, error) {
					return func(characterId uint32) (character.Model, error) {
						return character.NewModelBuilder().Build(), nil
					}
				}
			},
			wantPassed:        false,
			wantDetailsCount:  0,
			wantError:         true,
			wantErrorContains: "invalid condition",
		},
		{
			name:        "With decorator - add custom detail",
			characterId: 123,
			conditions:  []string{"jobId=100"},
			decorators: []model.Decorator[ValidationResult]{
				func(vr ValidationResult) ValidationResult {
					// Create a new validation result with the same character ID
					result := NewValidationResult(vr.CharacterId())

					// Add all the original details using AddResult
					// This is a workaround since we can't directly access the private fields
					// We'll add them as "passed" conditions with the original detail text
					for _, detail := range vr.Details() {
						// Extract the description part after the "Passed: " or "Failed: " prefix
						description := ""
						if strings.HasPrefix(detail, "Passed: ") {
							description = strings.TrimPrefix(detail, "Passed: ")
							result.AddResult(true, description)
						} else if strings.HasPrefix(detail, "Failed: ") {
							description = strings.TrimPrefix(detail, "Failed: ")
							result.AddResult(false, description)
						} else {
							// If there's no prefix, just add it as is
							result.AddResult(true, detail)
						}
					}

					// Add a custom detail
					result.AddResult(true, "Custom detail from decorator")
					return result
				},
			},
			setupMock: func(m *mock.ProcessorImpl) {
				m.GetByIdFunc = func(decorators ...model.Decorator[character.Model]) func(characterId uint32) (character.Model, error) {
					return func(characterId uint32) (character.Model, error) {
						return character.NewModelBuilder().
							SetJobId(100).
							Build(), nil
					}
				}
			},
			wantPassed:       true,
			wantDetailsCount: 2, // 1 from condition + 1 from decorator
			wantError:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock character processor
			mockCharProcessor := &mock.ProcessorImpl{}
			if tt.setupMock != nil {
				tt.setupMock(mockCharProcessor)
			}

			// Create a validation processor with the mock character processor
			processor := &ProcessorImpl{
				l:                  logger,
				ctx:                context.Background(),
				characterProcessor: mockCharProcessor,
			}

			// Call the Validate function with decorators
			result, err := processor.Validate(tt.decorators...)(tt.characterId, tt.conditions)

			// Check for expected errors
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
					return
				}
				if tt.wantErrorContains != "" && !strings.Contains(err.Error(), tt.wantErrorContains) {
					t.Errorf("Expected error containing '%s', got '%v'", tt.wantErrorContains, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check validation result
			if result.Passed() != tt.wantPassed {
				t.Errorf("Validation passed = %v, want %v", result.Passed(), tt.wantPassed)
			}

			if len(result.Details()) != tt.wantDetailsCount {
				t.Errorf("Validation details count = %v, want %v", len(result.Details()), tt.wantDetailsCount)
			}
		})
	}
}

// TestValidateConditions tests the condition validation logic directly
func TestValidateConditions(t *testing.T) {
	tests := []struct {
		name           string
		characterModel character.Model
		conditions     []string
		wantPassed     bool
		wantDetails    int
		wantError      bool
	}{
		{
			name: "All conditions pass",
			characterModel: character.NewModelBuilder().
				SetJobId(100).
				SetMeso(10000).
				SetMapId(2000).
				SetFame(50).
				Build(),
			conditions: []string{"jobId=100", "meso>=10000", "mapId=2000", "fame>=50"},
			wantPassed: true,
			wantDetails: 4,
			wantError:  false,
		},
		{
			name: "Some conditions fail",
			characterModel: character.NewModelBuilder().
				SetJobId(100).
				SetMeso(10000).
				SetMapId(2000).
				SetFame(50).
				Build(),
			conditions: []string{"jobId=100", "meso>=20000", "mapId=2000", "fame>=60"},
			wantPassed: false,
			wantDetails: 4,
			wantError:  false,
		},
		{
			name: "Invalid condition",
			characterModel: character.NewModelBuilder().Build(),
			conditions:     []string{"invalid=100"},
			wantPassed:     true, // Default is true, but error will be returned
			wantDetails:    0,
			wantError:      true,
		},
		{
			name: "Empty conditions",
			characterModel: character.NewModelBuilder().Build(),
			conditions:     []string{},
			wantPassed:     true, // No conditions means all pass
			wantDetails:    0,
			wantError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a validation result
			result := NewValidationResult(123)

			var err error
			// Process each condition
			for _, expr := range tt.conditions {
				var condition Condition
				condition, err = NewCondition(expr)
				if err != nil {
					break
				}

				// Evaluate the condition
				passed, description := condition.Evaluate(tt.characterModel)
				result.AddResult(passed, description)
			}

			// Check for expected errors
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check validation result
			if result.Passed() != tt.wantPassed {
				t.Errorf("Validation passed = %v, want %v", result.Passed(), tt.wantPassed)
			}

			if len(result.Details()) != tt.wantDetails {
				t.Errorf("Validation details count = %v, want %v", len(result.Details()), tt.wantDetails)
			}
		})
	}
}
