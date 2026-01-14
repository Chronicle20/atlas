package reactor

import (
	"atlas-reactors/reactor/data"
	"time"

	"github.com/Chronicle20/atlas-tenant"
)

type Model struct {
	tenant         tenant.Model
	id             uint32
	worldId        byte
	channelId      byte
	mapId          uint32
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

func (m Model) WorldId() byte {
	return m.worldId
}

func (m Model) ChannelId() byte {
	return m.channelId
}

func (m Model) MapId() uint32 {
	return m.mapId
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
