package party_quest

import (
	"fmt"
	"strconv"

	"github.com/google/uuid"
)

// Model represents a party quest instance for validation purposes
type Model struct {
	id         uuid.UUID
	customData map[string]string
}

// Id returns the instance ID
func (m Model) Id() uuid.UUID {
	return m.id
}

// CustomData returns the custom data map
func (m Model) CustomData() map[string]string {
	return m.customData
}

// GetCustomDataValue returns the value for a custom data key, or "" if not found
func (m Model) GetCustomDataValue(key string) string {
	if m.customData == nil {
		return ""
	}
	return m.customData[key]
}

// GetCustomDataInt returns the integer value for a custom data key, or 0 if not found or not numeric
func (m Model) GetCustomDataInt(key string) int {
	val := m.GetCustomDataValue(key)
	if val == "" {
		return 0
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}
	return n
}

// ModelBuilder provides a builder pattern for creating party quest models
type ModelBuilder struct {
	id         uuid.UUID
	customData map[string]string
}

// NewModelBuilder creates a new party quest model builder
func NewModelBuilder() *ModelBuilder {
	return &ModelBuilder{
		customData: make(map[string]string),
	}
}

// SetId sets the instance ID
func (b *ModelBuilder) SetId(id uuid.UUID) *ModelBuilder {
	b.id = id
	return b
}

// SetCustomData sets the custom data map
func (b *ModelBuilder) SetCustomData(data map[string]string) *ModelBuilder {
	b.customData = data
	return b
}

// Build creates a party quest model from the builder
func (b *ModelBuilder) Build() Model {
	return Model{
		id:         b.id,
		customData: b.customData,
	}
}

// StageStateRestModel represents stage state from the atlas-party-quests REST API
type StageStateRestModel struct {
	CustomData map[string]any `json:"customData,omitempty"`
}

// RestModel represents a party quest instance from the atlas-party-quests REST API
type RestModel struct {
	Id         uuid.UUID           `json:"-"`
	StageState StageStateRestModel `json:"stageState"`
}

func (r RestModel) GetName() string {
	return "instances"
}

func (r RestModel) GetID() string {
	return r.Id.String()
}

func (r *RestModel) SetID(idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid instance ID: %w", err)
	}
	r.Id = id
	return nil
}

// Extract transforms a RestModel into a domain Model
func Extract(r RestModel) (Model, error) {
	customData := make(map[string]string)
	for k, v := range r.StageState.CustomData {
		customData[k] = fmt.Sprintf("%v", v)
	}

	return NewModelBuilder().
		SetId(r.Id).
		SetCustomData(customData).
		Build(), nil
}
