package pet

import (
	"atlas-channel/pet/exclude"
	"errors"
	"time"
)

var ErrInvalidId = errors.New("pet id must be greater than 0")

type builder struct {
	id         uint32
	cashId     uint64
	templateId uint32
	name       string
	level      byte
	closeness  uint16
	fullness   byte
	expiration time.Time
	ownerId    uint32
	slot       int8
	x          int16
	y          int16
	stance     byte
	fh         int16
	excludes   []exclude.Model
	flag       uint16
	purchaseBy uint32
}

// NewBuilder creates a new builder instance with required fields
func NewBuilder(id uint32, cashId uint64, templateId uint32, name string) *builder {
	return &builder{
		id:         id,
		cashId:     cashId,
		templateId: templateId,
		name:       name,
		excludes:   make([]exclude.Model, 0),
	}
}

// CloneModel creates a builder initialized with the Model's values
func CloneModel(m Model) *builder {
	return &builder{
		id:         m.id,
		cashId:     m.cashId,
		templateId: m.templateId,
		name:       m.name,
		level:      m.level,
		closeness:  m.closeness,
		fullness:   m.fullness,
		expiration: m.expiration,
		ownerId:    m.ownerId,
		slot:       m.slot,
		x:          m.x,
		y:          m.y,
		stance:     m.stance,
		fh:         m.fh,
		excludes:   m.excludes,
		flag:       m.flag,
		purchaseBy: m.purchaseBy,
	}
}

func (b *builder) SetLevel(level byte) *builder {
	b.level = level
	return b
}

func (b *builder) SetCloseness(closeness uint16) *builder {
	b.closeness = closeness
	return b
}

func (b *builder) SetFullness(fullness byte) *builder {
	b.fullness = fullness
	return b
}

func (b *builder) SetExpiration(expiration time.Time) *builder {
	b.expiration = expiration
	return b
}

func (b *builder) SetOwnerID(ownerId uint32) *builder {
	b.ownerId = ownerId
	return b
}

func (b *builder) SetSlot(slot int8) *builder {
	b.slot = slot
	return b
}

func (b *builder) SetX(x int16) *builder {
	b.x = x
	return b
}

func (b *builder) SetY(y int16) *builder {
	b.y = y
	return b
}

func (b *builder) SetStance(stance byte) *builder {
	b.stance = stance
	return b
}

func (b *builder) SetExcludes(excludes []exclude.Model) *builder {
	b.excludes = excludes
	return b
}

func (b *builder) SetFoothold(fh int16) *builder {
	b.fh = fh
	return b
}

func (b *builder) SetFlag(flag uint16) *builder {
	b.flag = flag
	return b
}

func (b *builder) SetPurchaseBy(purchaseBy uint32) *builder {
	b.purchaseBy = purchaseBy
	return b
}

// Build creates a new Model instance with validation
func (b *builder) Build() (Model, error) {
	if b.id == 0 {
		return Model{}, ErrInvalidId
	}
	return Model{
		id:         b.id,
		cashId:     b.cashId,
		templateId: b.templateId,
		name:       b.name,
		level:      b.level,
		closeness:  b.closeness,
		fullness:   b.fullness,
		expiration: b.expiration,
		ownerId:    b.ownerId,
		slot:       b.slot,
		x:          b.x,
		y:          b.y,
		stance:     b.stance,
		fh:         b.fh,
		excludes:   b.excludes,
		flag:       b.flag,
		purchaseBy: b.purchaseBy,
	}, nil
}

// MustBuild creates a new Model instance, panicking on validation error
func (b *builder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
