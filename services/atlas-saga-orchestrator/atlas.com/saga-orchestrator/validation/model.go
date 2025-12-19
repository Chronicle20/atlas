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

// ConditionBuilder is used to safely construct Condition objects
type ConditionBuilder struct {
	conditionType ConditionType
	operator      Operator
	value         int
	itemId        *uint32
	err           error
}

// NewConditionBuilder creates a new condition builder
func NewConditionBuilder() *ConditionBuilder {
	return &ConditionBuilder{}
}

// SetType sets the condition type
func (b *ConditionBuilder) SetType(condType string) *ConditionBuilder {
	if b.err != nil {
		return b
	}

	switch ConditionType(condType) {
	case JobCondition, MesoCondition, MapCondition, FameCondition, ItemCondition:
		b.conditionType = ConditionType(condType)
	default:
		b.err = fmt.Errorf("unsupported condition type: %s", condType)
	}
	return b
}

// SetOperator sets the operator
func (b *ConditionBuilder) SetOperator(op string) *ConditionBuilder {
	if b.err != nil {
		return b
	}

	switch Operator(op) {
	case Equals, GreaterThan, LessThan, GreaterEqual, LessEqual:
		b.operator = Operator(op)
	default:
		b.err = fmt.Errorf("unsupported operator: %s", op)
	}
	return b
}

// SetValue sets the value
func (b *ConditionBuilder) SetValue(value int) *ConditionBuilder {
	if b.err != nil {
		return b
	}

	b.value = value
	return b
}

// SetItemId sets the item ID (only for item conditions)
func (b *ConditionBuilder) SetItemId(itemId uint32) *ConditionBuilder {
	if b.err != nil {
		return b
	}

	b.itemId = &itemId
	return b
}

// FromInput creates a condition builder from a ConditionInput
func (b *ConditionBuilder) FromInput(input ConditionInput) *ConditionBuilder {
	b.SetType(input.Type)
	b.SetOperator(input.Operator)
	b.SetValue(input.Value)

	if input.ItemId != 0 {
		b.SetItemId(input.ItemId)
	} else if ConditionType(input.Type) == ItemCondition {
		b.err = fmt.Errorf("itemId is required for item conditions")
	}

	return b
}

// Validate validates the builder state
func (b *ConditionBuilder) Validate() *ConditionBuilder {
	if b.err != nil {
		return b
	}

	// Check if condition type is set
	if b.conditionType == "" {
		b.err = fmt.Errorf("condition type is required")
		return b
	}

	// Check if operator is set
	if b.operator == "" {
		b.err = fmt.Errorf("operator is required")
		return b
	}

	// Check if itemId is set for item conditions
	if b.conditionType == ItemCondition && b.itemId == nil {
		b.err = fmt.Errorf("itemId is required for item conditions")
		return b
	}

	return b
}

// Build builds a Condition from the builder
func (b *ConditionBuilder) Build() (Condition, error) {
	b.Validate()

	if b.err != nil {
		return Condition{}, b.err
	}

	condition := Condition{
		conditionType: b.conditionType,
		operator:      b.operator,
		value:         b.value,
	}

	if b.itemId != nil {
		condition.itemId = *b.itemId
	}

	return condition, nil
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
