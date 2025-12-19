package reactor

import (
	"atlas-reactors/reactor/data"
	"github.com/Chronicle20/atlas-tenant"
	"time"
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

type ModelBuilder struct {
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

func NewModelBuilder(t tenant.Model, worldId byte, channelId byte, mapId uint32, classification uint32, name string) *ModelBuilder {
	return &ModelBuilder{
		tenant:         t,
		worldId:        worldId,
		channelId:      channelId,
		mapId:          mapId,
		classification: classification,
		name:           name,
		updateTime:     time.Now(),
	}
}

func NewFromModel(m Model) *ModelBuilder {
	return &ModelBuilder{
		tenant:         m.tenant,
		id:             m.Id(),
		worldId:        m.WorldId(),
		channelId:      m.ChannelId(),
		mapId:          m.MapId(),
		classification: m.Classification(),
		name:           m.Name(),
		data:           m.Data(),
		state:          m.State(),
		eventState:     m.EventState(),
		delay:          m.Delay(),
		direction:      m.Direction(),
		x:              m.X(),
		y:              m.Y(),
		updateTime:     m.UpdateTime(),
	}
}

func (b *ModelBuilder) Build() Model {
	return Model{
		tenant:         b.tenant,
		id:             b.id,
		worldId:        b.worldId,
		channelId:      b.channelId,
		mapId:          b.mapId,
		classification: b.classification,
		name:           b.name,
		data:           b.data,
		state:          b.state,
		eventState:     b.eventState,
		delay:          b.delay,
		direction:      b.direction,
		x:              b.x,
		y:              b.y,
		updateTime:     b.updateTime,
	}
}

func (b *ModelBuilder) SetState(state int8) *ModelBuilder {
	b.state = state
	return b
}

func (b *ModelBuilder) SetPosition(x int16, y int16) *ModelBuilder {
	b.x = x
	b.y = y
	return b
}

func (b *ModelBuilder) SetDelay(delay uint32) *ModelBuilder {
	b.delay = delay
	return b
}

func (b *ModelBuilder) SetDirection(direction byte) *ModelBuilder {
	b.direction = direction
	return b
}

func (b *ModelBuilder) Classification() uint32 {
	return b.classification
}

func (b *ModelBuilder) SetData(data data.Model) *ModelBuilder {
	b.data = data
	return b
}

func (b *ModelBuilder) UpdateTime() *ModelBuilder {
	b.updateTime = time.Now()
	return b
}

func (b *ModelBuilder) SetId(id uint32) *ModelBuilder {
	b.id = id
	return b
}
