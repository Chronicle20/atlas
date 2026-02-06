package saved_location

import (
	_map "github.com/Chronicle20/atlas-constants/map"
)

// RestModel is the JSON:API model for saved locations
type RestModel struct {
	Id           string  `json:"-"`
	CharacterId  uint32  `json:"characterId"`
	LocationType string  `json:"locationType"`
	MapId        _map.Id `json:"mapId"`
	PortalId     uint32  `json:"portalId"`
}

// GetName returns the resource type name
func (r RestModel) GetName() string {
	return "saved-locations"
}

// GetID returns the resource ID
func (r RestModel) GetID() string {
	return r.Id
}

// SetID sets the resource ID
func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// Extract converts RestModel to Model
func Extract(rm RestModel) (Model, error) {
	return NewBuilder().
		SetCharacterId(rm.CharacterId).
		SetLocationType(rm.LocationType).
		SetMapId(rm.MapId).
		SetPortalId(rm.PortalId).
		Build(), nil
}
