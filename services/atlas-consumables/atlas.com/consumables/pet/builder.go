package pet

import (
	"time"
)

type Builder struct {
	id              uint64
	inventoryItemId uint32
	templateId      uint32
	name            string
	level           byte
	closeness       uint16
	fullness        byte
	expiration      time.Time
	ownerId         uint32
	lead            bool
	slot            int8
	x               int16
	y               int16
	stance          byte
	fh              int16
}

func NewBuilder(id uint64, inventoryItemId, templateId uint32, name string) *Builder {
	return &Builder{
		id:              id,
		inventoryItemId: inventoryItemId,
		templateId:      templateId,
		name:            name,
	}
}

func (b *Builder) SetLevel(level byte) *Builder {
	b.level = level
	return b
}

func (b *Builder) SetCloseness(closeness uint16) *Builder {
	b.closeness = closeness
	return b
}

func (b *Builder) SetFullness(fullness byte) *Builder {
	b.fullness = fullness
	return b
}

func (b *Builder) SetExpiration(expiration time.Time) *Builder {
	b.expiration = expiration
	return b
}

func (b *Builder) SetOwnerID(ownerId uint32) *Builder {
	b.ownerId = ownerId
	return b
}

func (b *Builder) SetLead(lead bool) *Builder {
	b.lead = lead
	return b
}

func (b *Builder) SetSlot(slot int8) *Builder {
	b.slot = slot
	return b
}

func (b *Builder) SetX(x int16) *Builder {
	b.x = x
	return b
}

func (b *Builder) SetY(y int16) *Builder {
	b.y = y
	return b
}

func (b *Builder) SetStance(stance byte) *Builder {
	b.stance = stance
	return b
}

func (b *Builder) Build() Model {
	return Model{
		id:              b.id,
		inventoryItemId: b.inventoryItemId,
		templateId:      b.templateId,
		name:            b.name,
		level:           b.level,
		closeness:       b.closeness,
		fullness:        b.fullness,
		expiration:      b.expiration,
		ownerId:         b.ownerId,
		lead:            b.lead,
		slot:            b.slot,
		x:               b.x,
		y:               b.y,
		stance:          b.stance,
		fh:              b.fh,
	}
}

func (b *Builder) SetFoothold(fh int16) *Builder {
	b.fh = fh
	return b
}
