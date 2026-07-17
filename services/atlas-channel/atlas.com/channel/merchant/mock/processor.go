package mock

import (
	"atlas-channel/merchant"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	InFieldModelProviderFunc func(f field.Model) model.Provider[[]merchant.Model]
	ForEachInFieldFunc       func(f field.Model, o model.Operator[merchant.Model]) error
	GetVisitingShopFunc      func(characterId uint32) (merchant.Model, error)
	GetShopFunc              func(shopId string) (merchant.Model, error)
	GetByCharacterIdFunc     func(characterId uint32) ([]merchant.Model, error)
	PlaceShopFunc            func(f field.Model, characterId uint32, shopType byte, title string, permitItemId uint32, x int16, y int16) error
	OpenShopFunc             func(characterId uint32, shopId uuid.UUID) error
	CloseShopFunc            func(characterId uint32, shopId uuid.UUID) error
	EnterShopFunc            func(characterId uint32, shopId uuid.UUID) error
	ExitShopFunc             func(characterId uint32, shopId uuid.UUID) error
	SendMessageFunc          func(characterId uint32, shopId uuid.UUID, content string) error
	EnterMaintenanceFunc     func(characterId uint32, shopId uuid.UUID) error
	ExitMaintenanceFunc      func(characterId uint32, shopId uuid.UUID) error
	AddListingFunc           func(characterId uint32, shopId uuid.UUID, inventoryType byte, slot int16, quantity uint16, bundleSize uint16, pricePerBundle uint32) error
	RemoveListingFunc        func(characterId uint32, shopId uuid.UUID, listingIndex uint16) error
	PurchaseBundleFunc       func(characterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16) error
}

var _ merchant.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) InFieldModelProvider(f field.Model) model.Provider[[]merchant.Model] {
	if m.InFieldModelProviderFunc != nil {
		return m.InFieldModelProviderFunc(f)
	}
	return model.FixedProvider([]merchant.Model{})
}

func (m *ProcessorMock) ForEachInField(f field.Model, o model.Operator[merchant.Model]) error {
	if m.ForEachInFieldFunc != nil {
		return m.ForEachInFieldFunc(f, o)
	}
	return nil
}

func (m *ProcessorMock) GetVisitingShop(characterId uint32) (merchant.Model, error) {
	if m.GetVisitingShopFunc != nil {
		return m.GetVisitingShopFunc(characterId)
	}
	return merchant.Model{}, nil
}

func (m *ProcessorMock) GetShop(shopId string) (merchant.Model, error) {
	if m.GetShopFunc != nil {
		return m.GetShopFunc(shopId)
	}
	return merchant.Model{}, nil
}

func (m *ProcessorMock) GetByCharacterId(characterId uint32) ([]merchant.Model, error) {
	if m.GetByCharacterIdFunc != nil {
		return m.GetByCharacterIdFunc(characterId)
	}
	return nil, nil
}

func (m *ProcessorMock) PlaceShop(f field.Model, characterId uint32, shopType byte, title string, permitItemId uint32, x int16, y int16) error {
	if m.PlaceShopFunc != nil {
		return m.PlaceShopFunc(f, characterId, shopType, title, permitItemId, x, y)
	}
	return nil
}

func (m *ProcessorMock) OpenShop(characterId uint32, shopId uuid.UUID) error {
	if m.OpenShopFunc != nil {
		return m.OpenShopFunc(characterId, shopId)
	}
	return nil
}

func (m *ProcessorMock) CloseShop(characterId uint32, shopId uuid.UUID) error {
	if m.CloseShopFunc != nil {
		return m.CloseShopFunc(characterId, shopId)
	}
	return nil
}

func (m *ProcessorMock) EnterShop(characterId uint32, shopId uuid.UUID) error {
	if m.EnterShopFunc != nil {
		return m.EnterShopFunc(characterId, shopId)
	}
	return nil
}

func (m *ProcessorMock) ExitShop(characterId uint32, shopId uuid.UUID) error {
	if m.ExitShopFunc != nil {
		return m.ExitShopFunc(characterId, shopId)
	}
	return nil
}

func (m *ProcessorMock) SendMessage(characterId uint32, shopId uuid.UUID, content string) error {
	if m.SendMessageFunc != nil {
		return m.SendMessageFunc(characterId, shopId, content)
	}
	return nil
}

func (m *ProcessorMock) EnterMaintenance(characterId uint32, shopId uuid.UUID) error {
	if m.EnterMaintenanceFunc != nil {
		return m.EnterMaintenanceFunc(characterId, shopId)
	}
	return nil
}

func (m *ProcessorMock) ExitMaintenance(characterId uint32, shopId uuid.UUID) error {
	if m.ExitMaintenanceFunc != nil {
		return m.ExitMaintenanceFunc(characterId, shopId)
	}
	return nil
}

func (m *ProcessorMock) AddListing(characterId uint32, shopId uuid.UUID, inventoryType byte, slot int16, quantity uint16, bundleSize uint16, pricePerBundle uint32) error {
	if m.AddListingFunc != nil {
		return m.AddListingFunc(characterId, shopId, inventoryType, slot, quantity, bundleSize, pricePerBundle)
	}
	return nil
}

func (m *ProcessorMock) RemoveListing(characterId uint32, shopId uuid.UUID, listingIndex uint16) error {
	if m.RemoveListingFunc != nil {
		return m.RemoveListingFunc(characterId, shopId, listingIndex)
	}
	return nil
}

func (m *ProcessorMock) PurchaseBundle(characterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16) error {
	if m.PurchaseBundleFunc != nil {
		return m.PurchaseBundleFunc(characterId, shopId, listingIndex, bundleCount)
	}
	return nil
}
