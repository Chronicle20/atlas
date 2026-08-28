package reactor

import (
	"errors"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type Builder struct {
	id             uint32
	f              field.Model
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

func NewBuilder(f field.Model, classification uint32, name string) *Builder {
	return &Builder{
		f:              f,
		classification: classification,
		name:           name,
		updateTime:     time.Now(),
	}
}

func NewFromModel(m Model) *Builder {
	return &Builder{
		id:             m.Id(),
		f:              m.Field(),
		classification: m.Classification(),
		name:           m.Name(),
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
	if b.name == "" {
		return Model{}, errors.New("reactor name cannot be empty")
	}
	if b.classification == 0 {
		return Model{}, errors.New("reactor classification must be positive")
	}

	return Model{
		id:             b.id,
		f:              b.f,
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

func (b *Builder) UpdateTime() *Builder {
	b.updateTime = time.Now()
	return b
}

func (b *Builder) SetId(id uint32) *Builder {
	b.id = id
	return b
}

func (b *Builder) SetEventState(state byte) *Builder {
	b.eventState = state
	return b
}
