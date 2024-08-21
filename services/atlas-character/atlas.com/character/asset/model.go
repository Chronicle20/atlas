package asset

import (
	"github.com/Chronicle20/atlas-model/model"
)

type CharacterAssetCreator func(characterId uint32) InventoryAssetCreator

type InventoryAssetCreator func(inventoryId uint32, inventoryType int8) ItemCreator

type ItemCreator func(itemId uint32) Creator

type Creator func(quantity uint32) model.Provider[Asset]

type QuantityUpdater func(id uint32, quantity uint32) error

type Asset interface {
	Identifier
	Template
	Slottable
	Quantity
}

type Identifier interface {
	Id() uint32
}

type Template interface {
	ItemId() uint32
}

type Slottable interface {
	Slot() int16
}

type Quantity interface {
	Quantity() uint32
}

func NoOpSliceProvider() ([]Asset, error) {
	return make([]Asset, 0), nil
}

func NoOpQuantityUpdater(id uint32, quantity uint32) error {
	return nil
}
