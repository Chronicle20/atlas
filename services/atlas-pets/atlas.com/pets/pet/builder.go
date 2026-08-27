package pet

import (
	"atlas-pets/pet/exclude"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Builder struct {
	id                  uint32
	cashId              uint64
	templateId          uint32
	name                string
	level               byte
	closeness           uint16
	fullness            byte
	expiration          time.Time
	ownerId             uint32
	slot                int8
	excludes            []exclude.Model
	flag                uint16
	purchaseBy          uint32
	reviveTransactionId *uuid.UUID
}

func NewBuilder(id uint32, cashId uint64, templateId uint32, name string, ownerId uint32) *Builder {
	return &Builder{
		id:         id,
		cashId:     cashId,
		templateId: templateId,
		name:       name,
		level:      1,
		closeness:  0,
		fullness:   100,
		expiration: time.Now().Add(2160 * time.Hour),
		ownerId:    ownerId,
		slot:       -1,
		excludes:   make([]exclude.Model, 0),
		flag:       0,
		purchaseBy: ownerId,
	}
}

func Clone(m Model) *Builder {
	return NewBuilder(m.Id(), m.CashId(), m.TemplateId(), m.Name(), m.OwnerId()).
		SetLevel(m.Level()).
		SetCloseness(m.Closeness()).
		SetFullness(m.Fullness()).
		SetExpiration(m.Expiration()).
		SetSlot(m.Slot()).
		SetExcludes(m.Excludes()).
		SetFlag(m.Flag()).
		SetPurchaseBy(m.PurchaseBy()).
		SetReviveTransactionId(m.ReviveTransactionId())
}

func (b *Builder) SetLevel(level byte) *Builder {
	b.level = level
	return b
}

func (b *Builder) SetTemplateId(templateId uint32) *Builder {
	b.templateId = templateId
	return b
}

func (b *Builder) SetName(name string) *Builder {
	b.name = name
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

func (b *Builder) SetSlot(slot int8) *Builder {
	b.slot = slot
	return b
}

func (b *Builder) SetExcludes(excludes []exclude.Model) *Builder {
	b.excludes = excludes
	return b
}

func (b *Builder) SetFlag(flag uint16) *Builder {
	b.flag = flag
	return b
}

func (b *Builder) SetPurchaseBy(by uint32) *Builder {
	b.purchaseBy = by
	return b
}

func (b *Builder) SetReviveTransactionId(id *uuid.UUID) *Builder {
	b.reviveTransactionId = id
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.templateId == 0 {
		return Model{}, errors.New("templateId is required")
	}
	if b.ownerId == 0 {
		return Model{}, errors.New("ownerId is required")
	}
	if b.name == "" {
		return Model{}, errors.New("name is required")
	}
	if b.level < 1 || b.level > 30 {
		return Model{}, errors.New("level must be between 1 and 30")
	}
	if b.fullness > 100 {
		return Model{}, errors.New("fullness must be between 0 and 100")
	}
	if b.slot < -1 || b.slot > 2 {
		return Model{}, errors.New("slot must be -1 or between 0 and 2")
	}

	return Model{
		id:                  b.id,
		cashId:              b.cashId,
		templateId:          b.templateId,
		name:                b.name,
		level:               b.level,
		closeness:           b.closeness,
		fullness:            b.fullness,
		expiration:          b.expiration,
		ownerId:             b.ownerId,
		slot:                b.slot,
		excludes:            b.excludes,
		flag:                b.flag,
		purchaseBy:          b.purchaseBy,
		reviveTransactionId: b.reviveTransactionId,
	}, nil
}
