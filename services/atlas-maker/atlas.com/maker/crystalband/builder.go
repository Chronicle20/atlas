package crystalband

import (
	"errors"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

type Builder struct {
	tenantId      uuid.UUID
	minLevel      uint32
	maxLevel      uint32
	crystalItemId item.Id
	count         uint32
}

func NewBuilder(tenantId uuid.UUID) *Builder {
	return &Builder{tenantId: tenantId}
}

func (b *Builder) SetMinLevel(minLevel uint32) *Builder {
	b.minLevel = minLevel
	return b
}

func (b *Builder) SetMaxLevel(maxLevel uint32) *Builder {
	b.maxLevel = maxLevel
	return b
}

func (b *Builder) SetCrystalItemId(crystalItemId item.Id) *Builder {
	b.crystalItemId = crystalItemId
	return b
}

func (b *Builder) SetCount(count uint32) *Builder {
	b.count = count
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.tenantId == uuid.Nil {
		return Model{}, errors.New("crystalband: tenantId cannot be nil")
	}
	if b.crystalItemId == 0 {
		return Model{}, errors.New("crystalband: crystalItemId cannot be zero")
	}
	if b.maxLevel < b.minLevel {
		return Model{}, errors.New("crystalband: maxLevel cannot be less than minLevel")
	}
	if b.count == 0 {
		return Model{}, errors.New("crystalband: count cannot be zero")
	}
	return Model{
		tenantId:      b.tenantId,
		minLevel:      b.minLevel,
		maxLevel:      b.maxLevel,
		crystalItemId: b.crystalItemId,
		count:         b.count,
	}, nil
}
