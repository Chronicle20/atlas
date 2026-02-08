package gachapon

import (
	"errors"

	"github.com/google/uuid"
)

type Builder struct {
	tenantId       uuid.UUID
	id             string
	name           string
	npcIds         []uint32
	commonWeight   uint32
	uncommonWeight uint32
	rareWeight     uint32
}

func NewBuilder(tenantId uuid.UUID, id string) *Builder {
	return &Builder{tenantId: tenantId, id: id}
}

func (b *Builder) SetName(name string) *Builder {
	b.name = name
	return b
}

func (b *Builder) SetNpcIds(npcIds []uint32) *Builder {
	b.npcIds = npcIds
	return b
}

func (b *Builder) SetCommonWeight(w uint32) *Builder {
	b.commonWeight = w
	return b
}

func (b *Builder) SetUncommonWeight(w uint32) *Builder {
	b.uncommonWeight = w
	return b
}

func (b *Builder) SetRareWeight(w uint32) *Builder {
	b.rareWeight = w
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.tenantId == uuid.Nil {
		return Model{}, errors.New("tenantId cannot be nil")
	}
	if b.id == "" {
		return Model{}, errors.New("id cannot be empty")
	}
	return Model{
		tenantId:       b.tenantId,
		id:             b.id,
		name:           b.name,
		npcIds:         b.npcIds,
		commonWeight:   b.commonWeight,
		uncommonWeight: b.uncommonWeight,
		rareWeight:     b.rareWeight,
	}, nil
}
