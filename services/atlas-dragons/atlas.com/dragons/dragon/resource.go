package dragon

import (
	"strconv"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// RestModel is the JSON:API representation. The resource id is the OWNER
// character id: the dragon has no identity of its own.
type RestModel struct {
	Id               string     `json:"-"`
	OwnerCharacterId uint32     `json:"ownerCharacterId"`
	X                int32      `json:"x"`
	Y                int32      `json:"y"`
	Stance           byte       `json:"stance"`
	JobId            uint16     `json:"jobId"`
	WorldId          world.Id   `json:"worldId"`
	ChannelId        channel.Id `json:"channelId"`
	MapId            _map.Id    `json:"mapId"`
	Instance         uuid.UUID  `json:"instance"`
}

func (m RestModel) GetID() string {
	return m.Id
}

func (m *RestModel) SetID(idStr string) error {
	m.Id = idStr
	return nil
}

func (m RestModel) GetName() string {
	return "dragons"
}

func Transform(m Model) (RestModel, error) {
	f := m.Field()
	return RestModel{
		Id:               strconv.Itoa(int(m.OwnerCharacterId())),
		OwnerCharacterId: m.OwnerCharacterId(),
		X:                m.X(),
		Y:                m.Y(),
		Stance:           m.Stance(),
		JobId:            uint16(m.JobId()),
		WorldId:          f.WorldId(),
		ChannelId:        f.ChannelId(),
		MapId:            f.MapId(),
		Instance:         f.Instance(),
	}, nil
}
