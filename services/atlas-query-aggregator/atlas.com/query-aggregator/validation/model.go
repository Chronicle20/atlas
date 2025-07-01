package validation

import (
	"atlas-query-aggregator/character"
	"fmt"
	inventory2 "github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
	"regexp"
	"strconv"
	"strings"
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

// Condition represents a validation condition
type Condition struct {
	conditionType ConditionType
	operator      Operator
	value         int
	itemId        uint32 // Used for item conditions
}

// NewCondition creates a new condition from a string expression
func NewCondition(expression string) (Condition, error) {
	// Check for item condition first (item[ITEM_ID]>=QUANTITY)
	itemRegex := regexp.MustCompile(`^item\[(\d+)\](>=|<=|=|>|<)(\d+)$`)
	if matches := itemRegex.FindStringSubmatch(expression); matches != nil {
		itemId, err := strconv.ParseUint(matches[1], 10, 32)
		if err != nil {
			return Condition{}, fmt.Errorf("invalid item ID in condition: %s", matches[1])
		}

		var op Operator
		switch matches[2] {
		case ">=":
			op = GreaterEqual
		case "<=":
			op = LessEqual
		case "=":
			op = Equals
		case ">":
			op = GreaterThan
		case "<":
			op = LessThan
		}

		quantity, err := strconv.Atoi(matches[3])
		if err != nil {
			return Condition{}, fmt.Errorf("invalid quantity in condition: %s", matches[3])
		}

		return Condition{
			conditionType: ItemCondition,
			operator:      op,
			value:         quantity,
			itemId:        uint32(itemId),
		}, nil
	}

	// Parse standard expressions like "jobId=100", "meso>=10000", etc.
	var condType ConditionType
	var op Operator
	var val string

	// Check for >= or <= first
	if strings.Contains(expression, ">=") {
		parts := strings.Split(expression, ">=")
		if len(parts) != 2 {
			return Condition{}, fmt.Errorf("invalid condition format: %s", expression)
		}
		condType = ConditionType(parts[0])
		op = GreaterEqual
		val = parts[1]
	} else if strings.Contains(expression, "<=") {
		parts := strings.Split(expression, "<=")
		if len(parts) != 2 {
			return Condition{}, fmt.Errorf("invalid condition format: %s", expression)
		}
		condType = ConditionType(parts[0])
		op = LessEqual
		val = parts[1]
	} else if strings.Contains(expression, "=") {
		parts := strings.Split(expression, "=")
		if len(parts) != 2 {
			return Condition{}, fmt.Errorf("invalid condition format: %s", expression)
		}
		condType = ConditionType(parts[0])
		op = Equals
		val = parts[1]
	} else if strings.Contains(expression, ">") {
		parts := strings.Split(expression, ">")
		if len(parts) != 2 {
			return Condition{}, fmt.Errorf("invalid condition format: %s", expression)
		}
		condType = ConditionType(parts[0])
		op = GreaterThan
		val = parts[1]
	} else if strings.Contains(expression, "<") {
		parts := strings.Split(expression, "<")
		if len(parts) != 2 {
			return Condition{}, fmt.Errorf("invalid condition format: %s", expression)
		}
		condType = ConditionType(parts[0])
		op = LessThan
		val = parts[1]
	} else {
		return Condition{}, fmt.Errorf("invalid condition format: %s", expression)
	}

	// Validate condition type
	switch condType {
	case JobCondition, MesoCondition, MapCondition, FameCondition:
		// Valid condition type
	default:
		return Condition{}, fmt.Errorf("unsupported condition type: %s", condType)
	}

	// Parse value
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return Condition{}, fmt.Errorf("invalid value in condition: %s", val)
	}

	return Condition{
		conditionType: condType,
		operator:      op,
		value:         intVal,
	}, nil
}

// Evaluate evaluates the condition against a character model
// Note: ItemCondition is handled separately in the processor
func (c Condition) Evaluate(character character.Model) (bool, string) {
	var actualValue int
	var description string

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
	case ItemCondition:
		// For item conditions, we need to check the inventory
		itemQuantity := 0
		it, ok := inventory2.TypeFromItemId(item.Id(c.itemId))
		if !ok {
			return false, fmt.Sprintf("Invalid item ID: %d", c.itemId)
		}

		compartment := character.Inventory().CompartmentByType(it)
		for _, a := range compartment.Assets() {
			if a.TemplateId() == c.itemId {
				itemQuantity += int(a.Quantity())
			}
		}

		// Compare the item quantity with the expected value
		var itemResult bool
		switch c.operator {
		case Equals:
			itemResult = itemQuantity == c.value
		case GreaterThan:
			itemResult = itemQuantity > c.value
		case LessThan:
			itemResult = itemQuantity < c.value
		case GreaterEqual:
			itemResult = itemQuantity >= c.value
		case LessEqual:
			itemResult = itemQuantity <= c.value
		}

		description = fmt.Sprintf("Item %d quantity %s %d", c.itemId, c.operator, c.value)
		return itemResult, description
	default:
		return false, fmt.Sprintf("Unsupported condition type: %s", c.conditionType)
	}

	// Compare the actual value with the expected value based on the operator
	var result bool
	switch c.operator {
	case Equals:
		result = actualValue == c.value
	case GreaterThan:
		result = actualValue > c.value
	case LessThan:
		result = actualValue < c.value
	case GreaterEqual:
		result = actualValue >= c.value
	case LessEqual:
		result = actualValue <= c.value
	}

	return result, description
}

// ValidationResult represents the result of a validation
type ValidationResult struct {
	passed      bool
	details     []string
	characterId uint32
}

// NewValidationResult creates a new validation result
func NewValidationResult(characterId uint32) ValidationResult {
	return ValidationResult{
		passed:      true,
		details:     []string{},
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

// CharacterId returns the character ID that was validated
func (v ValidationResult) CharacterId() uint32 {
	return v.characterId
}

// AddResult adds a condition evaluation result to the validation result
func (v *ValidationResult) AddResult(passed bool, description string) {
	if !passed {
		v.passed = false
	}
	status := "Passed"
	if !passed {
		status = "Failed"
	}
	v.details = append(v.details, fmt.Sprintf("%s: %s", status, description))
}
