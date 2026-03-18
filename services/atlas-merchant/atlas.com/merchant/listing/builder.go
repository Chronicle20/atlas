package listing

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

func NewBuilder() *ModelBuilder {
	return &ModelBuilder{}
}

type ModelBuilder struct {
	id               uuid.UUID
	shopId           uuid.UUID
	itemId           uint32
	itemType         byte
	quantity         uint16
	bundleSize       uint16
	bundlesRemaining uint16
	pricePerBundle   uint32
	itemSnapshot     json.RawMessage
	displayOrder     uint16
	version          uint32
	listedAt         time.Time
}

func (b *ModelBuilder) SetId(id uuid.UUID) *ModelBuilder {
	b.id = id
	return b
}

func (b *ModelBuilder) SetShopId(shopId uuid.UUID) *ModelBuilder {
	b.shopId = shopId
	return b
}

func (b *ModelBuilder) SetItemId(itemId uint32) *ModelBuilder {
	b.itemId = itemId
	return b
}

func (b *ModelBuilder) SetItemType(itemType byte) *ModelBuilder {
	b.itemType = itemType
	return b
}

func (b *ModelBuilder) SetQuantity(quantity uint16) *ModelBuilder {
	b.quantity = quantity
	return b
}

func (b *ModelBuilder) SetBundleSize(bundleSize uint16) *ModelBuilder {
	b.bundleSize = bundleSize
	return b
}

func (b *ModelBuilder) SetBundlesRemaining(bundlesRemaining uint16) *ModelBuilder {
	b.bundlesRemaining = bundlesRemaining
	return b
}

func (b *ModelBuilder) SetPricePerBundle(pricePerBundle uint32) *ModelBuilder {
	b.pricePerBundle = pricePerBundle
	return b
}

func (b *ModelBuilder) SetItemSnapshot(itemSnapshot json.RawMessage) *ModelBuilder {
	b.itemSnapshot = itemSnapshot
	return b
}

func (b *ModelBuilder) SetDisplayOrder(displayOrder uint16) *ModelBuilder {
	b.displayOrder = displayOrder
	return b
}

func (b *ModelBuilder) SetVersion(version uint32) *ModelBuilder {
	b.version = version
	return b
}

func (b *ModelBuilder) SetListedAt(listedAt time.Time) *ModelBuilder {
	b.listedAt = listedAt
	return b
}

func (b *ModelBuilder) Build() (Model, error) {
	if b.id == uuid.Nil {
		return Model{}, errors.New("id is required")
	}
	if b.shopId == uuid.Nil {
		return Model{}, errors.New("shopId is required")
	}
	if b.pricePerBundle == 0 {
		return Model{}, errors.New("pricePerBundle must be at least 1")
	}
	if b.bundleSize == 0 {
		return Model{}, errors.New("bundleSize must be at least 1")
	}
	return Model{
		id:               b.id,
		shopId:           b.shopId,
		itemId:           b.itemId,
		itemType:         b.itemType,
		quantity:         b.quantity,
		bundleSize:       b.bundleSize,
		bundlesRemaining: b.bundlesRemaining,
		pricePerBundle:   b.pricePerBundle,
		itemSnapshot:     b.itemSnapshot,
		displayOrder:     b.displayOrder,
		version:          b.version,
		listedAt:         b.listedAt,
	}, nil
}

func Clone(m Model) *ModelBuilder {
	return &ModelBuilder{
		id:               m.id,
		shopId:           m.shopId,
		itemId:           m.itemId,
		itemType:         m.itemType,
		quantity:         m.quantity,
		bundleSize:       m.bundleSize,
		bundlesRemaining: m.bundlesRemaining,
		pricePerBundle:   m.pricePerBundle,
		itemSnapshot:     m.itemSnapshot,
		displayOrder:     m.displayOrder,
		version:          m.version,
		listedAt:         m.listedAt,
	}
}
