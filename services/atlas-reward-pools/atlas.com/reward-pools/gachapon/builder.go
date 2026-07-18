package gachapon

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// KindGachapon and KindIncubator are the closed union of valid Kind values
// for a gachapon machine: the classic tiered reward pool, and the Pigmy Egg
// incubator pool. Every comparison against a machine's Kind must reference
// one of these constants rather than a bare string literal.
const (
	KindGachapon  = "gachapon"
	KindIncubator = "incubator"
)

// DefaultKind is the Kind a gachapon machine reports when the builder's
// SetKind is never called — the classic tiered reward pool. Existing rows
// (seeded before Kind existed) and existing callers that never mention Kind
// must continue to read this value.
const DefaultKind = KindGachapon

type Builder struct {
	tenantId       uuid.UUID
	id             string
	name           string
	npcIds         []uint32
	commonWeight   uint32
	uncommonWeight uint32
	rareWeight     uint32
	kind           string
}

func NewBuilder(tenantId uuid.UUID, id string) *Builder {
	return &Builder{tenantId: tenantId, id: id, kind: DefaultKind}
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

func (b *Builder) SetKind(kind string) *Builder {
	b.kind = kind
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.tenantId == uuid.Nil {
		return Model{}, errors.New("tenantId cannot be nil")
	}
	if b.id == "" {
		return Model{}, errors.New("id cannot be empty")
	}
	if b.kind != KindGachapon && b.kind != KindIncubator {
		return Model{}, fmt.Errorf("gachapon: invalid kind %q", b.kind)
	}
	return Model{
		tenantId:       b.tenantId,
		id:             b.id,
		name:           b.name,
		npcIds:         b.npcIds,
		commonWeight:   b.commonWeight,
		uncommonWeight: b.uncommonWeight,
		rareWeight:     b.rareWeight,
		kind:           b.kind,
	}, nil
}
