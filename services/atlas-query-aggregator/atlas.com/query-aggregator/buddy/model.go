package buddy

import (
	"github.com/google/uuid"
)

// RestModel represents the REST API response for a buddy list
type RestModel struct {
	Id          uuid.UUID `json:"-"`
	TenantId    uuid.UUID `json:"-"`
	CharacterId uint32    `json:"characterId"`
	Capacity    byte      `json:"capacity"`
}

// Model represents the buddy list model used in query-aggregator
type Model struct {
	characterId uint32
	capacity    byte
}

// NewModel creates a new buddy list model
func NewModel(characterId uint32, capacity byte) Model {
	return Model{
		characterId: characterId,
		capacity:    capacity,
	}
}

// CharacterId returns the character ID
func (m Model) CharacterId() uint32 {
	return m.characterId
}

// Capacity returns the buddy list capacity
func (m Model) Capacity() byte {
	return m.capacity
}

// Extract converts a RestModel to a Model
func Extract(rm RestModel) (Model, error) {
	return Model{
		characterId: rm.CharacterId,
		capacity:    rm.Capacity,
	}, nil
}
