package reactor

import (
	"errors"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

var ErrInvalidId = errors.New("reactor id must be greater than 0")

type builder struct {
	id             uint32
	field          field.Model
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

func NewBuilder(field field.Model, classification uint32, name string) *builder {
	return &builder{
		field:          field,
		classification: classification,
		name:           name,
		updateTime:     time.Now(),
	}
}

func CloneModel(m Model) *builder {
	return &builder{
		id:             m.id,
		field:          m.field,
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
func NewFromModel(m Model) *builder {
	return CloneModel(m)
}

func (b *builder) SetId(id uint32) *builder {
	b.id = id
	return b
}

func (b *builder) SetState(state int8) *builder {
	b.state = state
	return b
}

func (b *builder) SetPosition(x int16, y int16) *builder {
	b.x = x
	b.y = y
	return b
}

func (b *builder) SetDelay(delay uint32) *builder {
	b.delay = delay
	return b
}

func (b *builder) SetDirection(direction byte) *builder {
	b.direction = direction
	return b
}

func (b *builder) SetEventState(state byte) *builder {
	b.eventState = state
	return b
}

func (b *builder) UpdateTime() *builder {
	b.updateTime = time.Now()
	return b
}

func (b *builder) Classification() uint32 {
	return b.classification
}

func (b *builder) Build() (Model, error) {
	if b.id == 0 {
		return Model{}, ErrInvalidId
	}
	return Model{
		id:             b.id,
		field:          b.field,
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

func (b *builder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
