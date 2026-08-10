// Package inventory is the atlas-inventory REST client atlas-trades stages
// through. It reads one compartment at a time and answers the single question
// PUT_ITEM asks: what asset is sitting in this (inventoryType, slot), and is it
// flagged untradeable?
//
// The asset projection is deliberately narrow — id, slot, templateId, quantity
// and flag. That is everything the stage DECISION needs: which asset is here,
// how many of it, and is it flagged untradeable.
//
// It carries no stat snapshot on purpose. The snapshot that follows the item
// through escrow is taken by the saga orchestrator during expansion of
// transfer_to_trade, at the last moment before release_from_character deletes
// the asset (see assetSnapshotFromCompartmentAsset). Taking a second one here
// would create a second source of truth that could disagree with it.
package inventory

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// Asset is one item occupying a compartment slot.
type Asset struct {
	id         asset.Id
	slot       slot.Position
	templateId item.Id
	quantity   asset.Quantity
	flag       uint16
}

func (a Asset) Id() asset.Id { return a.id }

func (a Asset) Slot() slot.Position { return a.slot }

func (a Asset) TemplateId() item.Id { return a.templateId }

func (a Asset) Quantity() asset.Quantity { return a.quantity }

// Flag is the asset's raw flag bitfield. Interpret it through
// libs/atlas-constants/asset's Flag constants rather than by literal.
func (a Asset) Flag() uint16 { return a.flag }

// NewAsset builds one asset view. Asset is a value type with no mutable state,
// so it needs no builder.
func NewAsset(id asset.Id, s slot.Position, templateId item.Id, quantity asset.Quantity, flag uint16) Asset {
	return Asset{id: id, slot: s, templateId: templateId, quantity: quantity, flag: flag}
}

// Model is one inventory compartment and the assets it holds.
type Model struct {
	id            uuid.UUID
	inventoryType inventory.Type
	capacity      uint32
	assets        []Asset
}

// NewModel builds one compartment view. Model is a value type with no mutable
// state, so it needs no builder; Extract is its production caller.
func NewModel(id uuid.UUID, inventoryType inventory.Type, capacity uint32, assets []Asset) Model {
	out := make([]Asset, len(assets))
	copy(out, assets)
	return Model{id: id, inventoryType: inventoryType, capacity: capacity, assets: out}
}

func (m Model) Id() uuid.UUID { return m.id }

func (m Model) Type() inventory.Type { return m.inventoryType }

func (m Model) Capacity() uint32 { return m.capacity }

// Assets returns a copy of the asset list, so a caller cannot write through the
// returned slice into the compartment's state.
func (m Model) Assets() []Asset {
	if m.assets == nil {
		return nil
	}
	out := make([]Asset, len(m.assets))
	copy(out, m.assets)
	return out
}

// FindBySlot returns the asset occupying the given slot position.
func (m Model) FindBySlot(s slot.Position) (Asset, bool) {
	for _, a := range m.assets {
		if a.slot == s {
			return a, true
		}
	}
	return Asset{}, false
}

// FindById returns the asset with the given id, wherever it currently sits.
//
// Asset id is the only STABLE handle on an item: a slot position is not, because
// atlas-inventory lets a player swap a reserved slot and re-keys the reservation
// to the destination (compartment/processor.go:490-507, SwapReservation). Callers
// that recorded a slot earlier must re-resolve through this before acting on it.
func (m Model) FindById(id asset.Id) (Asset, bool) {
	for _, a := range m.assets {
		if a.id == id {
			return a, true
		}
	}
	return Asset{}, false
}
