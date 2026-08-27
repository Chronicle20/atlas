package listing

import (
	"atlas-merchant/kafka/message/asset"
	"errors"
	"time"

	"github.com/google/uuid"
)

func NewBuilder() *Builder {
	return &Builder{}
}

type Builder struct {
	id               uuid.UUID
	shopId           uuid.UUID
	itemId           uint32
	itemType         byte
	quantity         uint16
	bundleSize       uint16
	bundlesRemaining uint16
	pricePerBundle   uint32
	itemSnapshot     asset.AssetData
	displayOrder     uint16
	version          uint32
	listedAt         time.Time
}

func (b *Builder) SetId(id uuid.UUID) *Builder {
	b.id = id
	return b
}

func (b *Builder) SetShopId(shopId uuid.UUID) *Builder {
	b.shopId = shopId
	return b
}

func (b *Builder) SetItemId(itemId uint32) *Builder {
	b.itemId = itemId
	return b
}

func (b *Builder) SetItemType(itemType byte) *Builder {
	b.itemType = itemType
	return b
}

func (b *Builder) SetQuantity(quantity uint16) *Builder {
	b.quantity = quantity
	return b
}

func (b *Builder) SetBundleSize(bundleSize uint16) *Builder {
	b.bundleSize = bundleSize
	return b
}

func (b *Builder) SetBundlesRemaining(bundlesRemaining uint16) *Builder {
	b.bundlesRemaining = bundlesRemaining
	return b
}

func (b *Builder) SetPricePerBundle(pricePerBundle uint32) *Builder {
	b.pricePerBundle = pricePerBundle
	return b
}

func (b *Builder) SetItemSnapshot(itemSnapshot asset.AssetData) *Builder {
	b.itemSnapshot = itemSnapshot
	return b
}

func (b *Builder) SetDisplayOrder(displayOrder uint16) *Builder {
	b.displayOrder = displayOrder
	return b
}

func (b *Builder) SetVersion(version uint32) *Builder {
	b.version = version
	return b
}

func (b *Builder) SetListedAt(listedAt time.Time) *Builder {
	b.listedAt = listedAt
	return b
}

func (b *Builder) Build() (Model, error) {
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

func Clone(m Model) *Builder {
	return &Builder{
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
