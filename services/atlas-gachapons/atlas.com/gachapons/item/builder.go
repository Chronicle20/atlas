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

func (b *Builder) Build() (Model, error) {
	if b.tenantId == uuid.Nil {
		return Model{}, errors.New("tenantId cannot be nil")
	}
	if !isValidTier(b.tier) {
		return Model{}, errors.New("tier must be one of: common, uncommon, rare")
	}
	return Model{
		tenantId:   b.tenantId,
		id:         b.id,
		gachaponId: b.gachaponId,
		itemId:     b.itemId,
		quantity:   b.quantity,
		tier:       b.tier,
	}, nil
}

func isValidTier(tier string) bool {
	return tier == "common" || tier == "uncommon" || tier == "rare"
}
