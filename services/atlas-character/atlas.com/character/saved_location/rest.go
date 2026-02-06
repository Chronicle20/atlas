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

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:           m.Id().String(),
		CharacterId:  m.CharacterId(),
		LocationType: m.LocationType(),
		MapId:        m.MapId(),
		PortalId:     m.PortalId(),
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return NewBuilder().
		SetCharacterId(rm.CharacterId).
		SetLocationType(rm.LocationType).
		SetMapId(rm.MapId).
		SetPortalId(rm.PortalId).
		Build(), nil
}
