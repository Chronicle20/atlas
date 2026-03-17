package mock

import (
	"atlas-merchant/listing"
	"atlas-merchant/shop"
	"encoding/json"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
)

var _ shop.Processor = (*ProcessorMock)(nil)

type ProcessorMock struct {
	GetByIdFunc              func(id uuid.UUID) (shop.Model, error)
	ByIdProviderFunc         func(id uuid.UUID) model.Provider[shop.Model]
	GetByCharacterIdFunc     func(characterId uint32) ([]shop.Model, error)
	GetByMapIdFunc           func(mapId uint32) ([]shop.Model, error)
	GetAllOpenFunc           func() ([]shop.Model, error)
	GetListingCountsFunc     func(shopIds []uuid.UUID) (map[uuid.UUID]int64, error)
	SearchListingsByItemIdFunc func(itemId uint32) ([]shop.ListingSearchResult, error)
	GetListingsFunc          func(shopId uuid.UUID) ([]listing.Model, error)
	CreateShopFunc           func(characterId uint32, shopType shop.ShopType, title string, mapId uint32, x int16, y int16, permitItemId uint32) (shop.Model, error)
	OpenShopFunc             func(shopId uuid.UUID) error
	EnterMaintenanceFunc     func(shopId uuid.UUID) error
	ExitMaintenanceFunc      func(shopId uuid.UUID) (bool, error)
	CloseShopFunc            func(shopId uuid.UUID, reason shop.CloseReason) error
	GetExpiredFunc           func() ([]shop.Model, error)
	AddListingFunc           func(shopId uuid.UUID, itemId uint32, itemType byte, bundleSize uint16, bundleCount uint16, pricePerBundle uint32, itemSnapshot json.RawMessage, flag uint16) (listing.Model, error)
	RemoveListingFunc        func(shopId uuid.UUID, listingIndex uint16) (listing.Model, error)
	UpdateListingFunc        func(shopId uuid.UUID, listingIndex uint16, pricePerBundle uint32, bundleSize uint16, bundleCount uint16) error
	EnterShopFunc            func(characterId uint32, shopId uuid.UUID) error
	ExitShopFunc             func(characterId uint32, shopId uuid.UUID) error
	EjectAllVisitorsFunc     func(shopId uuid.UUID) ([]uint32, error)
	GetVisitorsFunc          func(shopId uuid.UUID) ([]uint32, error)
	PurchaseBundleFunc       func(buyerCharacterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16) (shop.PurchaseResult, error)
	AddListingAndEmitFunc    func(shopId uuid.UUID, characterId uint32, itemId uint32, itemType byte, bundleSize uint16, bundleCount uint16, pricePerBundle uint32, itemSnapshot json.RawMessage, flag uint16, inventoryType byte, assetId uint32) (listing.Model, error)
	OpenShopAndEmitFunc      func(shopId uuid.UUID, characterId uint32) error
	CloseShopAndEmitFunc     func(shopId uuid.UUID, characterId uint32, reason shop.CloseReason) error
	EnterMaintenanceAndEmitFunc func(shopId uuid.UUID, characterId uint32) error
	ExitMaintenanceAndEmitFunc  func(shopId uuid.UUID, characterId uint32) error
	EnterShopAndEmitFunc     func(characterId uint32, shopId uuid.UUID) error
	ExitShopAndEmitFunc      func(characterId uint32, shopId uuid.UUID) error
	PurchaseBundleAndEmitFunc func(buyerCharacterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16, worldId world.Id) (shop.PurchaseResult, error)
}

func (m *ProcessorMock) GetById(id uuid.UUID) (shop.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(id)
	}
	return shop.Model{}, nil
}

func (m *ProcessorMock) ByIdProvider(id uuid.UUID) model.Provider[shop.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(id)
	}
	return func() (shop.Model, error) {
		return shop.Model{}, nil
	}
}

func (m *ProcessorMock) GetByCharacterId(characterId uint32) ([]shop.Model, error) {
	if m.GetByCharacterIdFunc != nil {
		return m.GetByCharacterIdFunc(characterId)
	}
	return []shop.Model{}, nil
}

func (m *ProcessorMock) GetByMapId(mapId uint32) ([]shop.Model, error) {
	if m.GetByMapIdFunc != nil {
		return m.GetByMapIdFunc(mapId)
	}
	return []shop.Model{}, nil
}

func (m *ProcessorMock) GetAllOpen() ([]shop.Model, error) {
	if m.GetAllOpenFunc != nil {
		return m.GetAllOpenFunc()
	}
	return []shop.Model{}, nil
}

func (m *ProcessorMock) GetListingCounts(shopIds []uuid.UUID) (map[uuid.UUID]int64, error) {
	if m.GetListingCountsFunc != nil {
		return m.GetListingCountsFunc(shopIds)
	}
	return make(map[uuid.UUID]int64), nil
}

func (m *ProcessorMock) SearchListingsByItemId(itemId uint32) ([]shop.ListingSearchResult, error) {
	if m.SearchListingsByItemIdFunc != nil {
		return m.SearchListingsByItemIdFunc(itemId)
	}
	return []shop.ListingSearchResult{}, nil
}

func (m *ProcessorMock) GetListings(shopId uuid.UUID) ([]listing.Model, error) {
	if m.GetListingsFunc != nil {
		return m.GetListingsFunc(shopId)
	}
	return []listing.Model{}, nil
}

func (m *ProcessorMock) CreateShop(characterId uint32, shopType shop.ShopType, title string, mapId uint32, x int16, y int16, permitItemId uint32) (shop.Model, error) {
	if m.CreateShopFunc != nil {
		return m.CreateShopFunc(characterId, shopType, title, mapId, x, y, permitItemId)
	}
	return shop.Model{}, nil
}

func (m *ProcessorMock) OpenShop(shopId uuid.UUID) error {
	if m.OpenShopFunc != nil {
		return m.OpenShopFunc(shopId)
	}
	return nil
}

func (m *ProcessorMock) EnterMaintenance(shopId uuid.UUID) error {
	if m.EnterMaintenanceFunc != nil {
		return m.EnterMaintenanceFunc(shopId)
	}
	return nil
}

func (m *ProcessorMock) ExitMaintenance(shopId uuid.UUID) (bool, error) {
	if m.ExitMaintenanceFunc != nil {
		return m.ExitMaintenanceFunc(shopId)
	}
	return false, nil
}

func (m *ProcessorMock) CloseShop(shopId uuid.UUID, reason shop.CloseReason) error {
	if m.CloseShopFunc != nil {
		return m.CloseShopFunc(shopId, reason)
	}
	return nil
}

func (m *ProcessorMock) GetExpired() ([]shop.Model, error) {
	if m.GetExpiredFunc != nil {
		return m.GetExpiredFunc()
	}
	return []shop.Model{}, nil
}

func (m *ProcessorMock) AddListing(shopId uuid.UUID, itemId uint32, itemType byte, bundleSize uint16, bundleCount uint16, pricePerBundle uint32, itemSnapshot json.RawMessage, flag uint16) (listing.Model, error) {
	if m.AddListingFunc != nil {
		return m.AddListingFunc(shopId, itemId, itemType, bundleSize, bundleCount, pricePerBundle, itemSnapshot, flag)
	}
	return listing.Model{}, nil
}

func (m *ProcessorMock) RemoveListing(shopId uuid.UUID, listingIndex uint16) (listing.Model, error) {
	if m.RemoveListingFunc != nil {
		return m.RemoveListingFunc(shopId, listingIndex)
	}
	return listing.Model{}, nil
}

func (m *ProcessorMock) UpdateListing(shopId uuid.UUID, listingIndex uint16, pricePerBundle uint32, bundleSize uint16, bundleCount uint16) error {
	if m.UpdateListingFunc != nil {
		return m.UpdateListingFunc(shopId, listingIndex, pricePerBundle, bundleSize, bundleCount)
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

func (m *ProcessorMock) EjectAllVisitors(shopId uuid.UUID) ([]uint32, error) {
	if m.EjectAllVisitorsFunc != nil {
		return m.EjectAllVisitorsFunc(shopId)
	}
	return nil, nil
}

func (m *ProcessorMock) GetVisitors(shopId uuid.UUID) ([]uint32, error) {
	if m.GetVisitorsFunc != nil {
		return m.GetVisitorsFunc(shopId)
	}
	return nil, nil
}

func (m *ProcessorMock) PurchaseBundle(buyerCharacterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16) (shop.PurchaseResult, error) {
	if m.PurchaseBundleFunc != nil {
		return m.PurchaseBundleFunc(buyerCharacterId, shopId, listingIndex, bundleCount)
	}
	return shop.PurchaseResult{}, nil
}

func (m *ProcessorMock) AddListingAndEmit(shopId uuid.UUID, characterId uint32, itemId uint32, itemType byte, bundleSize uint16, bundleCount uint16, pricePerBundle uint32, itemSnapshot json.RawMessage, flag uint16, inventoryType byte, assetId uint32) (listing.Model, error) {
	if m.AddListingAndEmitFunc != nil {
		return m.AddListingAndEmitFunc(shopId, characterId, itemId, itemType, bundleSize, bundleCount, pricePerBundle, itemSnapshot, flag, inventoryType, assetId)
	}
	return listing.Model{}, nil
}

func (m *ProcessorMock) OpenShopAndEmit(shopId uuid.UUID, characterId uint32) error {
	if m.OpenShopAndEmitFunc != nil {
		return m.OpenShopAndEmitFunc(shopId, characterId)
	}
	return nil
}

func (m *ProcessorMock) CloseShopAndEmit(shopId uuid.UUID, characterId uint32, reason shop.CloseReason) error {
	if m.CloseShopAndEmitFunc != nil {
		return m.CloseShopAndEmitFunc(shopId, characterId, reason)
	}
	return nil
}

func (m *ProcessorMock) EnterMaintenanceAndEmit(shopId uuid.UUID, characterId uint32) error {
	if m.EnterMaintenanceAndEmitFunc != nil {
		return m.EnterMaintenanceAndEmitFunc(shopId, characterId)
	}
	return nil
}

func (m *ProcessorMock) ExitMaintenanceAndEmit(shopId uuid.UUID, characterId uint32) error {
	if m.ExitMaintenanceAndEmitFunc != nil {
		return m.ExitMaintenanceAndEmitFunc(shopId, characterId)
	}
	return nil
}

func (m *ProcessorMock) EnterShopAndEmit(characterId uint32, shopId uuid.UUID) error {
	if m.EnterShopAndEmitFunc != nil {
		return m.EnterShopAndEmitFunc(characterId, shopId)
	}
	return nil
}

func (m *ProcessorMock) ExitShopAndEmit(characterId uint32, shopId uuid.UUID) error {
	if m.ExitShopAndEmitFunc != nil {
		return m.ExitShopAndEmitFunc(characterId, shopId)
	}
	return nil
}

func (m *ProcessorMock) PurchaseBundleAndEmit(buyerCharacterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16, worldId world.Id) (shop.PurchaseResult, error) {
	if m.PurchaseBundleAndEmitFunc != nil {
		return m.PurchaseBundleAndEmitFunc(buyerCharacterId, shopId, listingIndex, bundleCount, worldId)
	}
	return shop.PurchaseResult{}, nil
}
