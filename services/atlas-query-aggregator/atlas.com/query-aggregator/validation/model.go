package validation

import (
	"atlas-query-aggregator/character"
	"fmt"
	inventory2 "github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
)

// ConditionType represents the type of condition to validate
type ConditionType string

const (
	JobCondition           ConditionType = "jobId"
	MesoCondition          ConditionType = "meso"
	MapCondition           ConditionType = "mapId"
	FameCondition          ConditionType = "fame"
	ItemCondition          ConditionType = "item"
	GenderCondition        ConditionType = "gender"
	LevelCondition         ConditionType = "level"
	RebornsCondition       ConditionType = "reborns"
	DojoPointsCondition    ConditionType = "dojoPoints"
	VanquisherKillsCondition ConditionType = "vanquisherKills"
	GmLevelCondition       ConditionType = "gmLevel"
	GuildIdCondition       ConditionType = "guildId"
	GuildRankCondition     ConditionType = "guildRank"
	QuestStatusCondition   ConditionType = "questStatus"
	QuestProgressCondition ConditionType = "questProgress"
	UnclaimedMarriageGiftsCondition ConditionType = "hasUnclaimedMarriageGifts"
	StrengthCondition      ConditionType = "strength"
	DexterityCondition     ConditionType = "dexterity"
	IntelligenceCondition  ConditionType = "intelligence"
	LuckCondition          ConditionType = "luck"
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
	case JobCondition, MesoCondition, MapCondition, FameCondition, ItemCondition, GenderCondition, LevelCondition, RebornsCondition, DojoPointsCondition, VanquisherKillsCondition, GmLevelCondition, GuildIdCondition, GuildRankCondition, QuestStatusCondition, QuestProgressCondition, UnclaimedMarriageGiftsCondition, StrengthCondition, DexterityCondition, IntelligenceCondition, LuckCondition:
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


// Evaluate evaluates the condition against a character model
// Returns a structured ConditionResult with evaluation details
func (c Condition) Evaluate(character character.Model) ConditionResult {
	var actualValue int
	var passed bool
	var description string
	var itemId uint32

	// Get the actual value from the character model based on condition type
	switch c.conditionType {
	case JobCondition:
		actualValue = int(character.JobId())
		description = fmt.Sprintf("Job ID %s %d", c.operator, c.value)
	case MesoCondition:
		actualValue = int(character.Meso())
		description = fmt.Sprintf("Meso %s %d", c.operator, c.value)
	case MapCondition:
		actualValue = int(character.MapId())
		description = fmt.Sprintf("Map ID %s %d", c.operator, c.value)
	case FameCondition:
		actualValue = int(character.Fame())
		description = fmt.Sprintf("Fame %s %d", c.operator, c.value)
	case GenderCondition:
		actualValue = int(character.Gender())
		description = fmt.Sprintf("Gender %s %d", c.operator, c.value)
	case LevelCondition:
		actualValue = int(character.Level())
		description = fmt.Sprintf("Level %s %d", c.operator, c.value)
	case RebornsCondition:
		actualValue = int(character.Reborns())
		description = fmt.Sprintf("Reborns %s %d", c.operator, c.value)
	case DojoPointsCondition:
		actualValue = int(character.DojoPoints())
		description = fmt.Sprintf("Dojo Points %s %d", c.operator, c.value)
	case VanquisherKillsCondition:
		actualValue = int(character.VanquisherKills())
		description = fmt.Sprintf("Vanquisher Kills %s %d", c.operator, c.value)
	case GmLevelCondition:
		actualValue = character.GmLevel()
		description = fmt.Sprintf("GM Level %s %d", c.operator, c.value)
	case GuildIdCondition:
		// TODO: Implement guild ID validation when guild model is available
		actualValue = 0 // Placeholder - character.Guild().Id()
		description = fmt.Sprintf("Guild ID %s %d", c.operator, c.value)
	case GuildRankCondition:
		// TODO: Implement guild rank validation when guild model is available
		actualValue = 0 // Placeholder - character.Guild().Rank()
		description = fmt.Sprintf("Guild Rank %s %d", c.operator, c.value)
	case QuestStatusCondition:
		// TODO: Implement quest status validation when quest integration is available
		actualValue = 0 // Placeholder - will need quest service integration
		description = fmt.Sprintf("Quest Status %s %d", c.operator, c.value)
	case QuestProgressCondition:
		// TODO: Implement quest progress validation when quest integration is available
		actualValue = 0 // Placeholder - will need quest service integration
		description = fmt.Sprintf("Quest Progress %s %d", c.operator, c.value)
	case UnclaimedMarriageGiftsCondition:
		// TODO: Implement marriage gifts validation when marriage integration is available
		actualValue = 0 // Placeholder - will need marriage service integration
		description = fmt.Sprintf("Unclaimed Marriage Gifts %s %d", c.operator, c.value)
	case StrengthCondition:
		actualValue = int(character.Strength())
		description = fmt.Sprintf("Strength %s %d", c.operator, c.value)
	case DexterityCondition:
		actualValue = int(character.Dexterity())
		description = fmt.Sprintf("Dexterity %s %d", c.operator, c.value)
	case IntelligenceCondition:
		actualValue = int(character.Intelligence())
		description = fmt.Sprintf("Intelligence %s %d", c.operator, c.value)
	case LuckCondition:
		actualValue = int(character.Luck())
		description = fmt.Sprintf("Luck %s %d", c.operator, c.value)
	case ItemCondition:
		// For item conditions, we need to check the inventory
		itemQuantity := 0
		it, ok := inventory2.TypeFromItemId(item.Id(c.itemId))
		if !ok {
			return ConditionResult{
				Passed:      false,
				Description: fmt.Sprintf("Invalid item ID: %d", c.itemId),
				Type:        c.conditionType,
				Operator:    c.operator,
				Value:       c.value,
				ItemId:      c.itemId,
				ActualValue: 0,
			}
		}

		compartment := character.Inventory().CompartmentByType(it)
		for _, a := range compartment.Assets() {
			if a.TemplateId() == c.itemId {
				itemQuantity += int(a.Quantity())
			}
		}

		actualValue = itemQuantity
		itemId = c.itemId
		description = fmt.Sprintf("Item %d quantity %s %d", c.itemId, c.operator, c.value)
	default:
		return ConditionResult{
			Passed:      false,
			Description: fmt.Sprintf("Unsupported condition type: %s", c.conditionType),
			Type:        c.conditionType,
			Operator:    c.operator,
			Value:       c.value,
			ActualValue: 0,
		}
	}

	// Compare the actual value with the expected value based on the operator
	switch c.operator {
	case Equals:
		passed = actualValue == c.value
	case GreaterThan:
		passed = actualValue > c.value
	case LessThan:
		passed = actualValue < c.value
	case GreaterEqual:
		passed = actualValue >= c.value
	case LessEqual:
		passed = actualValue <= c.value
	}

	return ConditionResult{
		Passed:      passed,
		Description: description,
		Type:        c.conditionType,
		Operator:    c.operator,
		Value:       c.value,
		ItemId:      itemId,
		ActualValue: actualValue,
	}
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
