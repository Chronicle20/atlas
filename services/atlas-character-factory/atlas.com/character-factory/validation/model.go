package validation

import (
	"fmt"
)

// ConditionType represents the type of condition to validate
type ConditionType string

const (
	JobCondition  ConditionType = "jobId"
	MesoCondition ConditionType = "meso"
	MapCondition  ConditionType = "mapId"
	FameCondition ConditionType = "fame"
	ItemCondition ConditionType = "item"
)

// Operator represents the comparison operator in a condition
type Operator string

const (
	Equals       Operator = "="
	GreaterThan  Operator = ">"
	LessThan     Operator = "<"
	GreaterEqual Operator = ">="
	LessEqual    Operator = "<="
)

// ConditionInput represents the structured input for creating a condition
type ConditionInput struct {
	Type     string `json:"type"`             // e.g., "jobId", "meso", "item"
	Operator string `json:"operator"`         // e.g., "=", ">=", "<"
	Value    int    `json:"value"`            // Value or quantity
	ItemId   uint32 `json:"itemId,omitempty"` // Only for item checks
}

// ConditionResult represents the result of a condition evaluation
type ConditionResult struct {
	Passed      bool
	Description string
	Type        ConditionType
	Operator    Operator
	Value       int
	ItemId      uint32
	ActualValue int
}

// Condition represents a validation condition
type Condition struct {
	conditionType ConditionType
	operator      Operator
	value         int
	itemId        uint32 // Used for item conditions
}

// ValidationResult represents the result of a validation
type ValidationResult struct {
	passed      bool
	details     []string
	results     []ConditionResult
	characterId uint32
}

// NewValidationResult creates a new validation result
func NewValidationResult(characterId uint32) ValidationResult {
	return ValidationResult{
		passed:      true,
		details:     []string{},
		results:     []ConditionResult{},
		characterId: characterId,
	}
}

// Passed returns whether the validation passed
func (v ValidationResult) Passed() bool {
	return v.passed
}

// Details returns the details of the validation
func (v ValidationResult) Details() []string {
	return v.details
}

// Results returns the structured condition results
func (v ValidationResult) Results() []ConditionResult {
	return v.results
}

// CharacterId returns the character ID that was validated
func (v ValidationResult) CharacterId() uint32 {
	return v.characterId
}

// AddConditionResult adds a structured condition result to the validation result
func (v *ValidationResult) AddConditionResult(result ConditionResult) {
	if !result.Passed {
		v.passed = false
	}
	status := "Passed"
	if !result.Passed {
		status = "Failed"
	}
	v.details = append(v.details, fmt.Sprintf("%s: %s", status, result.Description))
	v.results = append(v.results, result)
}
