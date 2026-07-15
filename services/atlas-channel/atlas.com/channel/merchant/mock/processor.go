package mock

import (
	"atlas-channel/merchant"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
)

type ProcessorMock struct {
	InFieldModelProviderFunc func(f field.Model) model.Provider[[]merchant.Model]
	ForEachInFieldFunc       func(f field.Model, o model.Operator[merchant.Model]) error
	GetVisitingShopFunc      func(characterId uint32) (merchant.Model, error)
	GetShopFunc              func(shopId string) (merchant.Model, error)
	GetByCharacterIdFunc     func(characterId uint32) ([]merchant.Model, error)
	HasFrederickPendingFunc  func(characterId uint32) (bool, error)
	PlaceShopFunc            func(f field.Model, characterId uint32, shopType byte, title string, permitItemId uint32, x int16, y int16) error
	OpenShopFunc             func(characterId uint32, shopId uuid.UUID) error
	CloseShopFunc            func(characterId uint32, shopId uuid.UUID) error
	EnterShopFunc            func(characterId uint32, shopId uuid.UUID, visitorName string) error
	AddBlacklistFunc         func(characterId uint32, shopId uuid.UUID, name string, bannedCharacterId uint32) error
	RemoveBlacklistFunc      func(characterId uint32, shopId uuid.UUID, name string) error
	GetBlacklistFunc         func(shopId string) ([]string, error)
	GetVisitsFunc            func(shopId string) ([]merchant.VisitEntry, error)
	ExitShopFunc             func(characterId uint32, shopId uuid.UUID) error
	SendMessageFunc          func(characterId uint32, shopId uuid.UUID, content string) error
	EnterMaintenanceFunc     func(characterId uint32, shopId uuid.UUID) error
	ExitMaintenanceFunc      func(characterId uint32, shopId uuid.UUID) error
	WithdrawMesoFunc         func(characterId uint32, shopId uuid.UUID) error
	OrganizeListingsFunc     func(characterId uint32, shopId uuid.UUID) error
	AddListingFunc           func(characterId uint32, shopId uuid.UUID, inventoryType byte, slot int16, quantity uint16, bundleSize uint16, pricePerBundle uint32) error
	RemoveListingFunc        func(characterId uint32, shopId uuid.UUID, listingIndex uint16) error
	PurchaseBundleFunc       func(characterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16) error
	SearchListingsFunc       func(worldId world.Id, itemId uint32, descending bool) ([]merchant.SearchListing, error)
	GetTopSearchesFunc       func(worldId world.Id) ([]merchant.TopSearch, error)
	RecordItemSearchFunc     func(f field.Model, characterId uint32, itemId uint32) error
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

func (m *ProcessorMock) HasFrederickPending(characterId uint32) (bool, error) {
	if m.HasFrederickPendingFunc != nil {
		return m.HasFrederickPendingFunc(characterId)
	}
	return false, nil
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

func (m *ProcessorMock) EnterShop(characterId uint32, shopId uuid.UUID, visitorName string) error {
	if m.EnterShopFunc != nil {
		return m.EnterShopFunc(characterId, shopId, visitorName)
	}
	return nil
}

func (m *ProcessorMock) AddBlacklist(characterId uint32, shopId uuid.UUID, name string, bannedCharacterId uint32) error {
	if m.AddBlacklistFunc != nil {
		return m.AddBlacklistFunc(characterId, shopId, name, bannedCharacterId)
	}
	return nil
}

func (m *ProcessorMock) RemoveBlacklist(characterId uint32, shopId uuid.UUID, name string) error {
	if m.RemoveBlacklistFunc != nil {
		return m.RemoveBlacklistFunc(characterId, shopId, name)
	}
	return nil
}

func (m *ProcessorMock) GetBlacklist(shopId string) ([]string, error) {
	if m.GetBlacklistFunc != nil {
		return m.GetBlacklistFunc(shopId)
	}
	return nil, nil
}

func (m *ProcessorMock) GetVisits(shopId string) ([]merchant.VisitEntry, error) {
	if m.GetVisitsFunc != nil {
		return m.GetVisitsFunc(shopId)
	}
	return nil, nil
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

func (m *ProcessorMock) WithdrawMeso(characterId uint32, shopId uuid.UUID) error {
	if m.WithdrawMesoFunc != nil {
		return m.WithdrawMesoFunc(characterId, shopId)
	}
	return nil
}

func (m *ProcessorMock) OrganizeListings(characterId uint32, shopId uuid.UUID) error {
	if m.OrganizeListingsFunc != nil {
		return m.OrganizeListingsFunc(characterId, shopId)
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

func (m *ProcessorMock) SearchListings(worldId world.Id, itemId uint32, descending bool) ([]merchant.SearchListing, error) {
	if m.SearchListingsFunc != nil {
		return m.SearchListingsFunc(worldId, itemId, descending)
	}
	return nil, nil
}

func (m *ProcessorMock) GetTopSearches(worldId world.Id) ([]merchant.TopSearch, error) {
	if m.GetTopSearchesFunc != nil {
		return m.GetTopSearchesFunc(worldId)
	}
	return nil, nil
}

func (m *ProcessorMock) RecordItemSearch(f field.Model, characterId uint32, itemId uint32) error {
	if m.RecordItemSearchFunc != nil {
		return m.RecordItemSearchFunc(f, characterId, itemId)
	}
	return nil
}
