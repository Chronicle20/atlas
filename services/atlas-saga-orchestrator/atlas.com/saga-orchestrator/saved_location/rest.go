package saved_location

import (
	_map "github.com/Chronicle20/atlas-constants/map"
)

type RestModel struct {
	Id           string  `json:"-"`
	CharacterId  uint32  `json:"characterId"`
	LocationType string  `json:"locationType"`
	MapId        _map.Id `json:"mapId"`
	PortalId     uint32  `json:"portalId"`
}

func (r RestModel) GetName() string {
	return "saved-locations"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}
