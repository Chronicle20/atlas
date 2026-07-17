package teleport_rock

import (
	"strconv"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// RestModel is the read-side JSON:API resource: both lists, unpadded (wire
// padding to EmptyMapId is the packet codec's job, not the API's — PRD §5).
type RestModel struct {
	Id      string    `json:"-"`
	Regular []_map.Id `json:"regular"`
	Vip     []_map.Id `json:"vip"`
}

func (r RestModel) GetName() string {
	return "teleport-rock-maps"
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
		Id:      strconv.FormatUint(uint64(m.CharacterId()), 10),
		Regular: m.Regular(),
		Vip:     m.Vip(),
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	characterId, err := strconv.ParseUint(rm.Id, 10, 32)
	if err != nil {
		characterId = 0
	}
	return NewBuilder().
		SetCharacterId(uint32(characterId)).
		SetRegular(rm.Regular).
		SetVip(rm.Vip).
		Build(), nil
}
