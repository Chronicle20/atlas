package teleport_rock

import (
	"errors"
	"net/http"
	"strconv"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// RestModel is the read-side JSON:API resource: both lists, unpadded (wire
// padding to EmptyMapId is the packet codec's job, not the API's — PRD §5).
type RestModel struct {
	Id              string    `json:"-"`
	Regular         []_map.Id `json:"regular"`
	Vip             []_map.Id `json:"vip"`
	RegularCapacity int       `json:"regularCapacity"`
	VipCapacity     int       `json:"vipCapacity"`
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
		Id:              strconv.FormatUint(uint64(m.CharacterId()), 10),
		Regular:         m.Regular(),
		Vip:             m.Vip(),
		RegularCapacity: RegularCapacity,
		VipCapacity:     VipCapacity,
	}, nil
}

// AddMapInputRestModel is the POST body:
// {data:{type:"teleport-rock-maps", attributes:{list,mapId}}}.
type AddMapInputRestModel struct {
	Id    string `json:"-"`
	List  string `json:"list"`
	MapId uint32 `json:"mapId"`
}

func (r AddMapInputRestModel) GetName() string {
	return "teleport-rock-maps"
}

func (r AddMapInputRestModel) GetID() string {
	return r.Id
}

func (r *AddMapInputRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// statusForError maps a typed validation error (from Processor.Add/Remove) to
// the HTTP status the REST handlers write. It returns 0 for an unrecognized
// (infrastructure) error, signaling the caller to fall back to
// server.WriteErrorResponse.
func statusForError(err error) int {
	switch {
	case errors.Is(err, ErrMapNotAllowed):
		return http.StatusBadRequest
	case errors.Is(err, ErrListFull), errors.Is(err, ErrDuplicate):
		return http.StatusConflict
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	default:
		return 0
	}
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
