package condition

import (
	"errors"
)

// Builder is a builder for Model
type Builder struct {
	conditionType   string
	operator        string
	value           string
	referenceId     string
	step            string
	worldId         string
	channelId       string
	includeEquipped bool
}

// NewBuilder creates a new Builder
func NewBuilder() *Builder {
	return &Builder{}
}

// SetType sets the condition type
func (b *Builder) SetType(condType string) *Builder {
	b.conditionType = condType
	return b
}

// SetOperator sets the operator
func (b *Builder) SetOperator(op string) *Builder {
	b.operator = op
	return b
}

// SetValue sets the value
func (b *Builder) SetValue(value string) *Builder {
	b.value = value
	return b
}

// SetReferenceId sets the reference ID
func (b *Builder) SetReferenceId(referenceId string) *Builder {
	b.referenceId = referenceId
	return b
}

// SetStep sets the step
func (b *Builder) SetStep(step string) *Builder {
	b.step = step
	return b
}

// SetWorldId sets the worldId
func (b *Builder) SetWorldId(worldId string) *Builder {
	b.worldId = worldId
	return b
}

// SetChannelId sets the channelId
func (b *Builder) SetChannelId(channelId string) *Builder {
	b.channelId = channelId
	return b
}

// SetIncludeEquipped sets whether to include equipped items in item condition checks
func (b *Builder) SetIncludeEquipped(includeEquipped bool) *Builder {
	b.includeEquipped = includeEquipped
	return b
}

// Build builds the Model
func (b *Builder) Build() (Model, error) {
	if b.conditionType == "" {
		return Model{}, errors.New("condition type is required")
	}
	if b.operator == "" {
		return Model{}, errors.New("operator is required")
	}
	if b.value == "" {
		return Model{}, errors.New("value is required")
	}

	return Model{
		conditionType:   b.conditionType,
		operator:        b.operator,
		value:           b.value,
		referenceId:     b.referenceId,
		step:            b.step,
		worldId:         b.worldId,
		channelId:       b.channelId,
		includeEquipped: b.includeEquipped,
	}, nil
}
