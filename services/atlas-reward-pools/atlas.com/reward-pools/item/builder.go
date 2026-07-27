package item

import (
	"errors"

	"github.com/google/uuid"
)

type Builder struct {
	tenantId   uuid.UUID
	id         uint32
	gachaponId string
	itemId     uint32
	quantity   uint32
	tier       string
	weight     uint32
}

func NewBuilder(tenantId uuid.UUID, id uint32) *Builder {
	return &Builder{tenantId: tenantId, id: id}
}

func (b *Builder) SetGachaponId(gachaponId string) *Builder {
	b.gachaponId = gachaponId
	return b
}

func (b *Builder) SetItemId(itemId uint32) *Builder {
	b.itemId = itemId
	return b
}

func (b *Builder) SetQuantity(quantity uint32) *Builder {
	b.quantity = quantity
	return b
}

func (b *Builder) SetTier(tier string) *Builder {
	b.tier = tier
	return b
}

// SetWeight sets an optional explicit roll weight for weighted (e.g.
// incubator) reward pools. Callers that never invoke it leave weight at its
// zero value.
func (b *Builder) SetWeight(weight uint32) *Builder {
	b.weight = weight
	return b
}

// ErrInvalidTier is returned when a caller supplies a tier outside the valid
// set. Shared by Builder.Build (create path) and Processor.Update (patch
// path) so both enforce the same rule.
var ErrInvalidTier = errors.New("tier must be one of: common, uncommon, rare")

func (b *Builder) Build() (Model, error) {
	if b.tenantId == uuid.Nil {
		return Model{}, errors.New("tenantId cannot be nil")
	}
	if !isValidTier(b.tier) {
		return Model{}, ErrInvalidTier
	}
	return Model{
		tenantId:   b.tenantId,
		id:         b.id,
		gachaponId: b.gachaponId,
		itemId:     b.itemId,
		quantity:   b.quantity,
		tier:       b.tier,
		weight:     b.weight,
	}, nil
}

func isValidTier(tier string) bool {
	return tier == "common" || tier == "uncommon" || tier == "rare"
}
