package field

import (
	"atlas-maps/map/character"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// RestModel is one live field instance. Id is
// "{worldId}:{channelId}:{mapId}:{instanceId}".
type RestModel struct {
	Id             string     `json:"-"`
	WorldId        world.Id   `json:"worldId"`
	ChannelId      channel.Id `json:"channelId"`
	MapId          _map.Id    `json:"mapId"`
	InstanceId     uuid.UUID  `json:"instanceId"`
	CharacterCount uint32     `json:"characterCount"`
}

func (m RestModel) GetID() string {
	return m.Id
}

func (m RestModel) GetName() string {
	return "fields"
}

func (m *RestModel) SetID(idStr string) error {
	m.Id = idStr
	return nil
}

// Transform maps a character.FieldOccupancy domain model to its REST
// projection.
func Transform(m character.FieldOccupancy) (RestModel, error) {
	return RestModel{
		Id:             string(m.Field.Id()),
		WorldId:        m.Field.WorldId(),
		ChannelId:      m.Field.ChannelId(),
		MapId:          m.Field.MapId(),
		InstanceId:     m.Field.Instance(),
		CharacterCount: m.CharacterCount,
	}, nil
}

// TransformSlice maps a slice of character.FieldOccupancy domain models to
// their REST projections. Returns the first transform error encountered, if
// any.
func TransformSlice(ms []character.FieldOccupancy) ([]RestModel, error) {
	out := make([]RestModel, 0, len(ms))
	for _, m := range ms {
		rm, err := Transform(m)
		if err != nil {
			return nil, err
		}
		out = append(out, rm)
	}
	return out, nil
}
