package reactor

import (
	"atlas-reactors/reactor/data"
	"errors"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

type ModelBuilder struct {
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

func NewModelBuilder(t tenant.Model, f field.Model, classification uint32, name string) *ModelBuilder {
	return &ModelBuilder{
		tenant:         t,
		worldId:        f.WorldId(),
		channelId:      f.ChannelId(),
		mapId:          f.MapId(),
		instance:       f.Instance(),
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
		instance:       m.Instance(),
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

func (b *ModelBuilder) Build() (Model, error) {
	if b.classification == 0 {
		return Model{}, errors.New("classification is required")
	}
	return Model{
		tenant:         b.tenant,
		id:             b.id,
		worldId:        b.worldId,
		channelId:      b.channelId,
		mapId:          b.mapId,
		instance:       b.instance,
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
	}, nil
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

func (b *ModelBuilder) SetName(name string) *ModelBuilder {
	b.name = name
	return b
}

func (b *ModelBuilder) Name() string {
	return b.name
}

func (b *ModelBuilder) UpdateTime() *ModelBuilder {
	b.updateTime = time.Now()
	return b
}

func (b *ModelBuilder) SetId(id uint32) *ModelBuilder {
	b.id = id
	return b
}
