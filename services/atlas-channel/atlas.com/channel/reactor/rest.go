package reactor

import (
	"strconv"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
)

type RestModel struct {
	Id             uint32     `json:"-"`
	WorldId        world.Id   `json:"worldId"`
	ChannelId      channel.Id `json:"channelId"`
	MapId          _map.Id    `json:"mapId"`
	Instance       uuid.UUID  `json:"instance"`
	Classification uint32     `json:"classification"`
	Name           string     `json:"name"`
	State          int8       `json:"state"`
	EventState     byte       `json:"eventState"`
	X              int16      `json:"x"`
	Y              int16      `json:"y"`
	Delay          uint32     `json:"delay"`
	Direction      byte       `json:"direction"`
}

func (r RestModel) GetName() string {
	return "reactors"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		return err
	}

	r.Id = uint32(id)
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:             rm.Id,
		worldId:        rm.WorldId,
		channelId:      rm.ChannelId,
		mapId:          rm.MapId,
		instance:       rm.Instance,
		classification: rm.Classification,
		name:           rm.Name,
		state:          rm.State,
		eventState:     rm.EventState,
		delay:          rm.Delay,
		direction:      rm.Direction,
		x:              rm.X,
		y:              rm.Y,
	}, nil
}
