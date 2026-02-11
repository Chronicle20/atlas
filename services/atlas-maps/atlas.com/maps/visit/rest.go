package visit

import (
	"strconv"
	"time"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/jtumidanski/api2go/jsonapi"
)

const (
	Resource = "visits"
)

type RestModel struct {
	Id             string `json:"-"`
	CharacterId    uint32 `json:"characterId"`
	MapId          uint32 `json:"mapId"`
	FirstVisitedAt string `json:"firstVisitedAt"`
}

func (r RestModel) GetName() string {
	return Resource
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}

func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}

func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{}
}

func Transform(v Visit) (RestModel, error) {
	return RestModel{
		Id:             strconv.FormatUint(uint64(v.MapId()), 10),
		CharacterId:    v.CharacterId(),
		MapId:          uint32(v.MapId()),
		FirstVisitedAt: v.FirstVisitedAt().Format(time.RFC3339),
	}, nil
}

func ExtractMapId(rm RestModel) _map.Id {
	return _map.Id(rm.MapId)
}
