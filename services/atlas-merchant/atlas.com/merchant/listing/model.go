package listing

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Model struct {
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

func (m Model) Id() uuid.UUID {
	return m.id
}

func (m Model) ShopId() uuid.UUID {
	return m.shopId
}

func (m Model) ItemId() uint32 {
	return m.itemId
}

func (m Model) ItemType() byte {
	return m.itemType
}

func (m Model) Quantity() uint16 {
	return m.quantity
}

func (m Model) BundleSize() uint16 {
	return m.bundleSize
}

func (m Model) BundlesRemaining() uint16 {
	return m.bundlesRemaining
}

func (m Model) PricePerBundle() uint32 {
	return m.pricePerBundle
}

func (m Model) ItemSnapshot() json.RawMessage {
	return m.itemSnapshot
}

func (m Model) DisplayOrder() uint16 {
	return m.displayOrder
}

func (m Model) Version() uint32 {
	return m.version
}

func (m Model) ListedAt() time.Time {
	return m.listedAt
}
