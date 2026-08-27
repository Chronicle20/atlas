package location

import (
	"strconv"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	Resource = "characters/%d/location"
)

// RestModel mirrors the JSON:API shape returned by atlas-maps's
// GET /characters/{id}/location endpoint. Required no-op relationship
// stubs are implemented per the api2go contract (see libs/atlas-rest CLAUDE.md).
type RestModel struct {
	Id        uint32     `json:"-"`
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
	State     string     `json:"state"`
}

func (r RestModel) GetName() string { return "character-locations" }

func (r RestModel) GetID() string { return strconv.FormatUint(uint64(r.Id), 10) }

func (r *RestModel) SetID(s string) error {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(v)
	return nil
}

func (r *RestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

// Transform maps a Model to its RestModel wire representation.
func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:        m.characterId,
		WorldId:   m.worldId,
		ChannelId: m.channelId,
		MapId:     m.mapId,
		Instance:  m.instance,
		State:     string(m.state),
	}, nil
}
