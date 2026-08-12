package kite

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type Builder struct {
	id          uint32
	f           field.Model
	characterId uint32
	name        string
	templateId  uint32
	message     string
	x           int16
	y           int16
	createdAt   time.Time
}

func NewBuilder(id uint32, f field.Model, characterId uint32) *Builder {
	return &Builder{id: id, f: f, characterId: characterId}
}

func (b *Builder) SetName(name string) *Builder {
	b.name = name
	return b
}

func (b *Builder) SetTemplateId(templateId uint32) *Builder {
	b.templateId = templateId
	return b
}

func (b *Builder) SetMessage(message string) *Builder {
	b.message = message
	return b
}

func (b *Builder) SetPosition(x int16, y int16) *Builder {
	b.x = x
	b.y = y
	return b
}

func (b *Builder) SetCreatedAt(t time.Time) *Builder {
	b.createdAt = t
	return b
}

func (b *Builder) Build() Model {
	return Model{
		id:          b.id,
		f:           b.f,
		characterId: b.characterId,
		name:        b.name,
		templateId:  b.templateId,
		message:     b.message,
		x:           b.x,
		y:           b.y,
		createdAt:   b.createdAt,
	}
}
