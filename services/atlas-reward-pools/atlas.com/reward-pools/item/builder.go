package item

import (
	"atlas-reward-pools/gachapon"
	"errors"

	"github.com/google/uuid"
)

type Builder struct {
	tenantId    uuid.UUID
	id          uint32
	gachaponId  string
	itemId      uint32
	quantity    uint32
	tier        string
	weight      uint32
	commodityId uint32
	// kind is the owning pool's Kind, recorded via SetKind purely for
	// Build-time validation. It is NOT part of Model and is never persisted
	// — the kind lives on the pool row, not the item row.
	kind string
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

// SetCommodityId sets the cash shop commodity (serial number) this entry
// awards. Required for cash-surprise entries; other kinds leave it 0.
func (b *Builder) SetCommodityId(commodityId uint32) *Builder {
	b.commodityId = commodityId
	return b
}

// SetKind records the owning pool's kind so Build can apply kind-specific
// validation. It is NOT persisted — the kind lives on the pool row.
func (b *Builder) SetKind(kind string) *Builder {
	b.kind = kind
	return b
}

// ErrInvalidTier is returned when a caller supplies a tier outside the valid
// set. Shared by Builder.Build (create path) and Processor.Update (patch
// path) so both enforce the same rule.
var ErrInvalidTier = errors.New("tier must be one of: common, uncommon, rare")

// ErrCommodityIdRequired is returned when a cash-surprise pool entry omits
// its commodity id. Such an entry can be rolled but never granted, so it is
// rejected at write time rather than failing silently at open time.
var ErrCommodityIdRequired = errors.New("commodityId is required for cash-surprise pool entries")

func (b *Builder) Build() (Model, error) {
	if b.tenantId == uuid.Nil {
		return Model{}, errors.New("tenantId cannot be nil")
	}
	if !isValidTier(b.tier) {
		return Model{}, ErrInvalidTier
	}
	if b.kind == gachapon.KindCashSurprise && b.commodityId == 0 {
		return Model{}, ErrCommodityIdRequired
	}
	return Model{
		tenantId:    b.tenantId,
		id:          b.id,
		gachaponId:  b.gachaponId,
		itemId:      b.itemId,
		quantity:    b.quantity,
		tier:        b.tier,
		weight:      b.weight,
		commodityId: b.commodityId,
	}, nil
}

func isValidTier(tier string) bool {
	return tier == "common" || tier == "uncommon" || tier == "rare"
}
