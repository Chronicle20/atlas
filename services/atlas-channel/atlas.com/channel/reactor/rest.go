package reactor

import (
	"strconv"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
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

// Transform converts the domain Model into the wire RestModel.
//
// UpdateTime is deliberately not carried: RestModel has no field for it, so
// Extract can never restore it. A Model round-tripped through Transform ->
// Extract therefore loses UpdateTime (see rest_test.go).
func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:             m.id,
		WorldId:        m.field.WorldId(),
		ChannelId:      m.field.ChannelId(),
		MapId:          m.field.MapId(),
		Instance:       m.field.Instance(),
		Classification: m.classification,
		Name:           m.name,
		State:          m.state,
		EventState:     m.eventState,
		X:              m.x,
		Y:              m.y,
		Delay:          m.delay,
		Direction:      m.direction,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:             rm.Id,
		field:          field.NewBuilder(rm.WorldId, rm.ChannelId, rm.MapId).SetInstance(rm.Instance).Build(),
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
