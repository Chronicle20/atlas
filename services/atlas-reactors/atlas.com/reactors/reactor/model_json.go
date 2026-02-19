package reactor

import (
	"atlas-reactors/reactor/data"
	"encoding/json"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

type modelJSON struct {
	Tenant         *tenant.Model `json:"tenant"`
	Id             uint32        `json:"id"`
	WorldId        world.Id      `json:"worldId"`
	ChannelId      channel.Id    `json:"channelId"`
	MapId          _map.Id       `json:"mapId"`
	Instance       uuid.UUID     `json:"instance"`
	Classification uint32        `json:"classification"`
	Name           string        `json:"name"`
	Data           data.Model    `json:"data"`
	State          int8          `json:"state"`
	EventState     byte          `json:"eventState"`
	Delay          uint32        `json:"delay"`
	Direction      byte          `json:"direction"`
	X              int16         `json:"x"`
	Y              int16         `json:"y"`
	UpdateTime     time.Time     `json:"updateTime"`
}

func (m Model) MarshalJSON() ([]byte, error) {
	t := m.tenant
	return json.Marshal(modelJSON{
		Tenant:         &t,
		Id:             m.id,
		WorldId:        m.worldId,
		ChannelId:      m.channelId,
		MapId:          m.mapId,
		Instance:       m.instance,
		Classification: m.classification,
		Name:           m.name,
		Data:           m.data,
		State:          m.state,
		EventState:     m.eventState,
		Delay:          m.delay,
		Direction:      m.direction,
		X:              m.x,
		Y:              m.y,
		UpdateTime:     m.updateTime,
	})
}

func (m *Model) UnmarshalJSON(b []byte) error {
	var j modelJSON
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}
	if j.Tenant != nil {
		m.tenant = *j.Tenant
	}
	m.id = j.Id
	m.worldId = j.WorldId
	m.channelId = j.ChannelId
	m.mapId = j.MapId
	m.instance = j.Instance
	m.classification = j.Classification
	m.name = j.Name
	m.data = j.Data
	m.state = j.State
	m.eventState = j.EventState
	m.delay = j.Delay
	m.direction = j.Direction
	m.x = j.X
	m.y = j.Y
	m.updateTime = j.UpdateTime
	return nil
}
