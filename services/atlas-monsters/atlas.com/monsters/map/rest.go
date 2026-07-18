package _map

import (
	"strconv"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	mapconst "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type RestModel struct {
	Id string `json:"-"`
}

func (m RestModel) GetID() string {
	return m.Id
}

func (m RestModel) GetName() string {
	return "characters"
}

func (m *RestModel) SetID(idStr string) error {
	m.Id = idStr
	return nil
}

func Extract(rm RestModel) (uint32, error) {
	id, err := strconv.ParseUint(rm.Id, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(id), nil
}

// LocationRestModel is the JSON:API projection of atlas-maps'
// GET /characters/{characterId}/location response.
type LocationRestModel struct {
	Id        uint32      `json:"-"`
	WorldId   world.Id    `json:"worldId"`
	ChannelId channel.Id  `json:"channelId"`
	MapId     mapconst.Id `json:"mapId"`
	Instance  uuid.UUID   `json:"instance"`
}

func (r LocationRestModel) GetName() string {
	return "character-locations"
}

func (r LocationRestModel) GetID() string {
	return strconv.FormatUint(uint64(r.Id), 10)
}

func (r *LocationRestModel) SetID(s string) error {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(v)
	return nil
}

// ExtractLocation converts the REST projection to a field.Model.
func ExtractLocation(rm LocationRestModel) (field.Model, error) {
	return field.NewBuilder(rm.WorldId, rm.ChannelId, rm.MapId).SetInstance(rm.Instance).Build(), nil
}
