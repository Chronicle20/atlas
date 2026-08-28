package reactor

import (
	"atlas-reactors/reactor/data"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Builder struct {
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

func NewBuilder(t tenant.Model, f field.Model, classification uint32, name string) *Builder {
	return &Builder{
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

func NewFromModel(m Model) *Builder {
	return &Builder{
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

func (b *Builder) Build() (Model, error) {
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

func (b *Builder) SetState(state int8) *Builder {
	b.state = state
	return b
}

func (b *Builder) SetPosition(x int16, y int16) *Builder {
	b.x = x
	b.y = y
	return b
}

func (b *Builder) SetDelay(delay uint32) *Builder {
	b.delay = delay
	return b
}

func (b *Builder) SetDirection(direction byte) *Builder {
	b.direction = direction
	return b
}

func (b *Builder) Classification() uint32 {
	return b.classification
}

func (b *Builder) Field() field.Model {
	return field.NewBuilder(b.worldId, b.channelId, b.mapId).SetInstance(b.instance).Build()
}

func (b *Builder) SetData(data data.Model) *Builder {
	b.data = data
	return b
}

func (b *Builder) SetName(name string) *Builder {
	b.name = name
	return b
}

func (b *Builder) Name() string {
	return b.name
}

func (b *Builder) UpdateTime() *Builder {
	b.updateTime = time.Now()
	return b
}

func (b *Builder) SetId(id uint32) *Builder {
	b.id = id
	return b
}
