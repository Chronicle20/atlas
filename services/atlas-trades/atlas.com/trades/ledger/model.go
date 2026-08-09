package ledger

import (
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// Item is one asset a side gave, as recorded at settlement time. assetId and
// referenceId are optional: only identity-bearing assets (equips, pets, cash)
// have them, so the getters report presence rather than handing back a pointer
// a caller could write through.
type Item struct {
	itemId         item.Id
	quantity       asset.Quantity
	assetId        asset.Id
	hasAssetId     bool
	referenceId    uint32
	hasReferenceId bool
}

// NewItem builds one recorded item. assetId and referenceId are nil for assets
// with no identity of their own (plain stackable consumables and etc items).
// Item is a value type with no mutable state, so it needs no builder.
func NewItem(itemId item.Id, quantity asset.Quantity, assetId *asset.Id, referenceId *uint32) Item {
	i := Item{itemId: itemId, quantity: quantity}
	if assetId != nil {
		i.assetId = *assetId
		i.hasAssetId = true
	}
	if referenceId != nil {
		i.referenceId = *referenceId
		i.hasReferenceId = true
	}
	return i
}

func (i Item) ItemId() item.Id { return i.itemId }

func (i Item) Quantity() asset.Quantity { return i.quantity }

// AssetId returns the asset's identity and whether it has one.
func (i Item) AssetId() (asset.Id, bool) { return i.assetId, i.hasAssetId }

// ReferenceId returns the asset's equip/pet/cash reference and whether it has
// one.
func (i Item) ReferenceId() (uint32, bool) { return i.referenceId, i.hasReferenceId }

// SideModel is one participant's recorded contribution. mesoDelivered is what
// the character actually received after the counterparty's tax, so
// mesoStaged/mesoTax describe what this side gave and mesoDelivered what it
// got (FR-7.1).
//
// Meso stays a plain uint32: libs/atlas-constants has no meso type.
type SideModel struct {
	id            uuid.UUID
	characterId   character.Id
	characterName string
	mesoStaged    uint32
	mesoTax       uint32
	mesoDelivered uint32
	items         []Item
}

func (s SideModel) Id() uuid.UUID { return s.id }

func (s SideModel) CharacterId() character.Id { return s.characterId }

func (s SideModel) CharacterName() string { return s.characterName }

func (s SideModel) MesoStaged() uint32 { return s.mesoStaged }

func (s SideModel) MesoTax() uint32 { return s.mesoTax }

func (s SideModel) MesoDelivered() uint32 { return s.mesoDelivered }

// Items returns a copy of the recorded items, so a caller cannot write through
// the returned slice into the side's state.
func (s SideModel) Items() []Item {
	if s.items == nil {
		return nil
	}
	out := make([]Item, len(s.items))
	copy(out, s.items)
	return out
}

// Model is one settled trade: the immutable ledger entry plus both sides.
//
// tenantId is stamped by the administrator on write and by Make on read; a
// Model that came straight out of the Builder and was never persisted carries
// uuid.Nil there, matching how id behaves.
type Model struct {
	id            uuid.UUID
	tenantId      uuid.UUID
	transactionId uuid.UUID
	f             field.Model
	roomType      byte
	settledAt     time.Time
	sides         []SideModel
}

func (m Model) Id() uuid.UUID { return m.id }

func (m Model) TenantId() uuid.UUID { return m.tenantId }

// TransactionId is the settlement saga's transaction id, unique per tenant
// (FR-5.7).
func (m Model) TransactionId() uuid.UUID { return m.transactionId }

// Field is where the trade settled. The instance is not recorded, so a Model
// read back from the ledger always carries uuid.Nil for it.
func (m Model) Field() field.Model { return m.f }

// RoomType is the miniroom type byte — miniroom.Trade (3) or
// miniroom.CashTrade (6).
func (m Model) RoomType() byte { return m.roomType }

func (m Model) SettledAt() time.Time { return m.settledAt }

// Sides returns a copy of the two sides, so a caller cannot write through the
// returned slice into the entry's state.
//
// ORDERING: sides read back from the ledger are ordered by character id, and
// items within a side by item id. That is a determinism guarantee, NOT a role
// guarantee — the ledger has no column recording which side owned the room, so
// Sides()[0] is the lower character id and says nothing about who initiated the
// trade. Match on CharacterId, never on position.
func (m Model) Sides() []SideModel {
	if m.sides == nil {
		return nil
	}
	out := make([]SideModel, len(m.sides))
	copy(out, m.sides)
	return out
}

// Make converts a persisted Entry (with its Sides and their Items preloaded)
// into an immutable Model. The error return is never non-nil today; it exists
// so Make satisfies model.Transformer and can be handed to model.SliceMap.
func Make(e Entry) (Model, error) {
	b := NewBuilder(e.TransactionId, field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).Build(), e.RoomType).
		SetId(e.Id).
		SetTenantId(e.TenantId).
		SetSettledAt(e.SettledAt)

	for _, s := range e.Sides {
		items := make([]Item, 0, len(s.Items))
		for _, i := range s.Items {
			items = append(items, NewItem(i.ItemId, i.Quantity, i.AssetId, i.ReferenceId))
		}
		b.addSideWithId(s.Id, s.CharacterId, s.CharacterName, s.MesoStaged, s.MesoTax, s.MesoDelivered, items)
	}
	return b.Build(), nil
}
