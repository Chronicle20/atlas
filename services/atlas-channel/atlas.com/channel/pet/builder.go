package pet

import (
	"atlas-channel/pet/exclude"
	"errors"
	"time"
)

var (
	ErrInvalidId = errors.New("pet id must be greater than 0")
)

type modelBuilder struct {
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

// NewModelBuilder creates a new builder instance with required fields
func NewModelBuilder(id uint32, cashId uint64, templateId uint32, name string) *modelBuilder {
	return &modelBuilder{
		id:         id,
		cashId:     cashId,
		templateId: templateId,
		name:       name,
		excludes:   make([]exclude.Model, 0),
	}
}

// CloneModel creates a builder initialized with the Model's values
func CloneModel(m Model) *modelBuilder {
	return &modelBuilder{
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

func (b *modelBuilder) SetLevel(level byte) *modelBuilder {
	b.level = level
	return b
}

func (b *modelBuilder) SetCloseness(closeness uint16) *modelBuilder {
	b.closeness = closeness
	return b
}

func (b *modelBuilder) SetFullness(fullness byte) *modelBuilder {
	b.fullness = fullness
	return b
}

func (b *modelBuilder) SetExpiration(expiration time.Time) *modelBuilder {
	b.expiration = expiration
	return b
}

func (b *modelBuilder) SetOwnerID(ownerId uint32) *modelBuilder {
	b.ownerId = ownerId
	return b
}

func (b *modelBuilder) SetSlot(slot int8) *modelBuilder {
	b.slot = slot
	return b
}

func (b *modelBuilder) SetX(x int16) *modelBuilder {
	b.x = x
	return b
}

func (b *modelBuilder) SetY(y int16) *modelBuilder {
	b.y = y
	return b
}

func (b *modelBuilder) SetStance(stance byte) *modelBuilder {
	b.stance = stance
	return b
}

func (b *modelBuilder) SetExcludes(excludes []exclude.Model) *modelBuilder {
	b.excludes = excludes
	return b
}

func (b *modelBuilder) SetFoothold(fh int16) *modelBuilder {
	b.fh = fh
	return b
}

func (b *modelBuilder) SetFlag(flag uint16) *modelBuilder {
	b.flag = flag
	return b
}

func (b *modelBuilder) SetPurchaseBy(purchaseBy uint32) *modelBuilder {
	b.purchaseBy = purchaseBy
	return b
}

// Build creates a new Model instance with validation
func (b *modelBuilder) Build() (Model, error) {
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
func (b *modelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic(err)
	}
	return m
}
