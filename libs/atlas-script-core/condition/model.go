package condition

import (
	"strconv"
)

// Model represents a condition that can be evaluated
type Model struct {
	conditionType   string
	operator        string
	value           string
	referenceId     string // String from JSON, will be converted to uint32 when needed
	step            string
	worldId         string // String from JSON, will be resolved from context for mapCapacity
	channelId       string // String from JSON, will be resolved from context for mapCapacity
	includeEquipped bool   // For item conditions: also check equipped items
}

// Type returns the condition type
func (c Model) Type() string {
	return c.conditionType
}

// Operator returns the operator
func (c Model) Operator() string {
	return c.operator
}

// Value returns the value
func (c Model) Value() string {
	return c.value
}

// ReferenceId returns the reference ID as uint32
// Note: This method does NOT support context references. Use ReferenceIdRaw() for that.
func (c Model) ReferenceId() uint32 {
	if c.referenceId == "" {
		return 0
	}
	// Convert string to uint32
	id, err := strconv.ParseUint(c.referenceId, 10, 32)
	if err != nil {
		return 0
	}
	return uint32(id)
}

// ReferenceIdRaw returns the raw reference ID string (may contain context reference like {context.itemId})
func (c Model) ReferenceIdRaw() string {
	return c.referenceId
}

// Step returns the step for quest progress
func (c Model) Step() string {
	return c.step
}

// WorldId returns the worldId (as string, may contain context reference)
func (c Model) WorldId() string {
	return c.worldId
}

// ChannelId returns the channelId (as string, may contain context reference)
func (c Model) ChannelId() string {
	return c.channelId
}

// IncludeEquipped returns whether to include equipped items in item condition checks
func (c Model) IncludeEquipped() bool {
	return c.includeEquipped
}

// Evaluator is the interface for evaluating conditions
type Evaluator interface {
	// EvaluateCondition evaluates a condition for a character
	EvaluateCondition(characterId uint32, condition Model) (bool, error)
}
