package reactor

import (
	"errors"
	"github.com/Chronicle20/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"time"
)

var (
	ErrInvalidId = errors.New("reactor id must be greater than 0")
)

type modelBuilder struct {
	id             uint32
	worldId        world.Id
	channelId      channel.Id
	mapId          _map.Id
	classification uint32
	name           string
	state          int8
	eventState     byte
	delay          uint32
	direction      byte
	x              int16
	y              int16
	updateTime     time.Time
}

func NewModelBuilder(worldId world.Id, channelId channel.Id, mapId _map.Id, classification uint32, name string) *modelBuilder {
	return &modelBuilder{
		worldId:        worldId,
		channelId:      channelId,
		mapId:          mapId,
		classification: classification,
		name:           name,
		updateTime:     time.Now(),
	}
}

func CloneModel(m Model) *modelBuilder {
	return &modelBuilder{
		id:             m.id,
		worldId:        m.worldId,
		channelId:      m.channelId,
		mapId:          m.mapId,
		classification: m.classification,
		name:           m.name,
		state:          m.state,
		eventState:     m.eventState,
		delay:          m.delay,
		direction:      m.direction,
		x:              m.x,
		y:              m.y,
		updateTime:     m.updateTime,
	}
}

// NewFromModel is an alias for CloneModel for backward compatibility
func NewFromModel(m Model) *modelBuilder {
	return CloneModel(m)
}

func (b *modelBuilder) SetId(id uint32) *modelBuilder {
	b.id = id
	return b
}

func (b *modelBuilder) SetState(state int8) *modelBuilder {
	b.state = state
	return b
}

func (b *modelBuilder) SetPosition(x int16, y int16) *modelBuilder {
	b.x = x
	b.y = y
	return b
}

func (b *modelBuilder) SetDelay(delay uint32) *modelBuilder {
	b.delay = delay
	return b
}

func (b *modelBuilder) SetDirection(direction byte) *modelBuilder {
	b.direction = direction
	return b
}

func (b *modelBuilder) SetEventState(state byte) *modelBuilder {
	b.eventState = state
	return b
}

func (b *modelBuilder) UpdateTime() *modelBuilder {
	b.updateTime = time.Now()
	return b
}

func (b *modelBuilder) Classification() uint32 {
	return b.classification
}

func (b *modelBuilder) Build() (Model, error) {
	if b.id == 0 {
		return Model{}, ErrInvalidId
	}
	return Model{
		id:             b.id,
		worldId:        b.worldId,
		channelId:      b.channelId,
		mapId:          b.mapId,
		classification: b.classification,
		name:           b.name,
		state:          b.state,
		eventState:     b.eventState,
		delay:          b.delay,
		direction:      b.direction,
		x:              b.x,
		y:              b.y,
		updateTime:     b.updateTime,
	}, nil
}

func (b *modelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
