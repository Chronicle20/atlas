package location

import (
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

// RestModel is the JSON:API projection of a character's last-known location.
type RestModel struct {
	Id        uint32     `json:"-"`
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map.Id    `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
}

// GetName returns the JSON:API resource type.
func (r RestModel) GetName() string {
	return "character-locations"
}

// GetID returns the JSON:API resource id (the character id).
func (r RestModel) GetID() string {
	return strconv.FormatUint(uint64(r.Id), 10)
}

// SetID parses the JSON:API resource id back into the model.
func (r *RestModel) SetID(s string) error {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(v)
	return nil
}

// SetToOneReferenceID is a no-op required by api2go's interface.
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

// SetToManyReferenceIDs is a no-op required by api2go's interface.
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// Transform maps a domain Model to its REST projection.
func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:        m.CharacterId(),
		WorldId:   m.WorldId(),
		ChannelId: m.ChannelId(),
		MapId:     m.MapId(),
		Instance:  m.Instance(),
	}, nil
}
