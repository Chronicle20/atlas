package field

import (
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
