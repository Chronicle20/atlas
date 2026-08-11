package escrow

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// ItemBuilder assembles an ItemModel. The three fields a row cannot be without
// — its own id, the room it backs and the owner it must be returned to — are
// constructor arguments rather than setters, because a row missing any of them
// is unreturnable and the reconciler would have nothing to act on.
//
// The snapshot is a single optional setter rather than one setter per stat. The
// per-stat form it replaced was the shape of the bug: adding a field to the
// asset meant remembering to add a setter AND to call it at every construction
// site, and the cash, expiry and pet fields were simply never added.
type ItemBuilder struct {
	m ItemModel
}

func NewItemBuilder(id uuid.UUID, roomId uuid.UUID, ownerId character.Id) *ItemBuilder {
	return &ItemBuilder{m: ItemModel{id: id, roomId: roomId, ownerId: ownerId}}
}

func (b *ItemBuilder) SetTradeSlot(v byte) *ItemBuilder { b.m.tradeSlot = v; return b }

// SetSource records where the item came from. The source SLOT rides on the
// snapshot (AssetSnapshot.Slot), so it is not a parameter here.
func (b *ItemBuilder) SetSource(inventoryType inventory.Type, assetId asset.Id) *ItemBuilder {
	b.m.sourceInventoryType = inventoryType
	b.m.assetId = assetId
	return b
}

func (b *ItemBuilder) SetSnapshot(v sharedsaga.AssetSnapshot) *ItemBuilder {
	b.m.snapshot = v
	return b
}

func (b *ItemBuilder) Build() ItemModel { return b.m }
