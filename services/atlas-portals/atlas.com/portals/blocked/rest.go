package blocked

import (
	"fmt"
)

// RestModel represents a blocked portal in JSON:API format
type RestModel struct {
	Id          string `json:"-"`
	CharacterId uint32 `json:"characterId"`
	MapId       uint32 `json:"mapId"`
	PortalId    uint32 `json:"portalId"`
}

// GetName returns the resource type name for JSON:API
func (r RestModel) GetName() string {
	return "blocked-portals"
}

// GetID returns the resource ID for JSON:API
func (r RestModel) GetID() string {
	return r.Id
}

// SetID sets the resource ID for JSON:API
func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// Transform converts a domain model to a REST model
func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:          fmt.Sprintf("%d:%d", m.MapId(), m.PortalId()),
		CharacterId: m.CharacterId(),
		MapId:       m.MapId(),
		PortalId:    m.PortalId(),
	}, nil
}
