package drop

import (
	"fmt"
	"strconv"
	"strings"
)

// RestModel is a JSON API representation of the reactor drop
type RestModel struct {
	Id        string `json:"-"`
	ReactorId uint32 `json:"-"`
	ItemId    uint32 `json:"itemId"`
	QuestId   uint32 `json:"questId,omitempty"`
	Chance    uint32 `json:"chance"`
}

// GetID to satisfy jsonapi.MarshalIdentifier interface
func (r RestModel) GetID() string {
	return r.Id
}

// SetID to satisfy jsonapi.UnmarshalIdentifier interface
func (r *RestModel) SetID(id string) error {
	r.Id = id
	// Parse the ID to extract reactorId, itemId, and optionally questId
	// Format: "reactorId:itemId" or "reactorId:itemId:questId"
	parts := strings.Split(id, ":")
	if len(parts) >= 2 {
		reactorId, err := strconv.ParseUint(parts[0], 10, 32)
		if err == nil {
			r.ReactorId = uint32(reactorId)
		}
		itemId, err := strconv.ParseUint(parts[1], 10, 32)
		if err == nil {
			r.ItemId = uint32(itemId)
		}
		if len(parts) >= 3 {
			questId, err := strconv.ParseUint(parts[2], 10, 32)
			if err == nil {
				r.QuestId = uint32(questId)
			}
		}
	}
	return nil
}

// GetName to satisfy jsonapi.EntityNamer interface
func (r RestModel) GetName() string {
	return "drops"
}

// GenerateID creates the composite ID for the drop
func GenerateID(reactorId, itemId, questId uint32) string {
	if questId > 0 {
		return fmt.Sprintf("%d:%d:%d", reactorId, itemId, questId)
	}
	return fmt.Sprintf("%d:%d", reactorId, itemId)
}

// Transform converts a Model to a RestModel
func Transform(model Model) (RestModel, error) {
	return RestModel{
		Id:        GenerateID(model.ReactorId(), model.ItemId(), model.QuestId()),
		ReactorId: model.ReactorId(),
		ItemId:    model.ItemId(),
		QuestId:   model.QuestId(),
		Chance:    model.Chance(),
	}, nil
}

// Extract converts a RestModel to domain Model fields
// Note: This returns individual fields since Builder requires tenantId
func Extract(rm RestModel) (reactorId, itemId, questId, chance uint32) {
	return rm.ReactorId, rm.ItemId, rm.QuestId, rm.Chance
}
