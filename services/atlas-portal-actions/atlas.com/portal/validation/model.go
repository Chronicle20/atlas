package validation

// ConditionInput represents a condition for validation
type ConditionInput struct {
	Type            string `json:"type"`
	Operator        string `json:"operator"`
	Value           int    `json:"value"`
	ReferenceId     uint32 `json:"referenceId,omitempty"`
	Step            string `json:"step,omitempty"`
	WorldId         byte   `json:"worldId,omitempty"`
	ChannelId       byte   `json:"channelId,omitempty"`
	IncludeEquipped bool   `json:"includeEquipped,omitempty"`
}

// ValidationResult represents the result of a validation
type ValidationResult struct {
	characterId uint32
	passed      bool
}

// NewValidationResult creates a new validation result
func NewValidationResult(characterId uint32, passed bool) ValidationResult {
	return ValidationResult{
		characterId: characterId,
		passed:      passed,
	}
}

// CharacterId returns the character ID that was validated
func (v ValidationResult) CharacterId() uint32 {
	return v.characterId
}

// Passed returns whether the validation passed
func (v ValidationResult) Passed() bool {
	return v.passed
}
