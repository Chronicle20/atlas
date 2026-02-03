package reactor

import (
	"atlas-reactors/reactor/data"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

type Model struct {
	tenant         tenant.Model
	id             uint32
	worldId        world.Id
	channelId      channel.Id
	mapId          _map.Id
	instance       uuid.UUID
	classification uint32
	name           string
	data           data.Model
	state          int8
	eventState     byte
	delay          uint32
	direction      byte
	x              int16
	y              int16
	updateTime     time.Time
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) WorldId() world.Id {
	return m.worldId
}

func (m Model) ChannelId() channel.Id {
	return m.channelId
}

func (m Model) MapId() _map.Id {
	return m.mapId
}

func (m Model) Instance() uuid.UUID {
	return m.instance
}

func (m Model) Field() field.Model {
	return field.NewBuilder(m.worldId, m.channelId, m.mapId).SetInstance(m.instance).Build()
}

func (m Model) Classification() uint32 {
	return m.classification
}

func (m Model) Name() string {
	return m.name
}

func (m Model) State() int8 {
	return m.state
}

func (m Model) EventState() byte {
	return m.eventState
}

func (m Model) Delay() uint32 {
	return m.delay
}

func (m Model) Direction() byte {
	return m.direction
}

func (m Model) X() int16 {
	return m.x
}

func (m Model) Y() int16 {
	return m.y
}

func (m Model) UpdateTime() time.Time {
	return m.updateTime
}

func (m Model) Data() data.Model {
	return m.data
}

func (m Model) Tenant() tenant.Model {
	return m.tenant
}
