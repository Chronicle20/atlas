package mock

import (
	message "atlas-merchant/kafka/message"
	"atlas-merchant/kafka/message/asset"
	"atlas-merchant/listing"
	"atlas-merchant/visit"
	"atlas-merchant/shop"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var _ shop.Processor = (*ProcessorMock)(nil)

type ProcessorMock struct {
	WithTransactionFunc        func(tx *gorm.DB) shop.Processor
	GetByIdFunc                func(id uuid.UUID) (shop.Model, error)
	ByIdProviderFunc           func(id uuid.UUID) model.Provider[shop.Model]
	GetByCharacterIdFunc       func(characterId uint32) ([]shop.Model, error)
	GetByCharacterIdPagedFunc  func(characterId uint32, page model.Page) (model.Paged[shop.Model], error)
	GetByFieldFunc             func(worldId world.Id, channelId channel.Id, mapId uint32, instanceId uuid.UUID) ([]shop.Model, error)
	GetByFieldPagedFunc        func(worldId world.Id, channelId channel.Id, mapId uint32, instanceId uuid.UUID, page model.Page) (model.Paged[shop.Model], error)
	GetAllOpenPagedFunc        func(page model.Page) (model.Paged[shop.Model], error)
	GetListingCountsFunc       func(shopIds []uuid.UUID) (map[uuid.UUID]int64, error)
	SearchListingsByItemIdPagedFunc func(criteria shop.ListingSearchCriteria, page model.Page) (model.Paged[shop.ListingSearchResult], error)
	GetListingsFunc            func(shopId uuid.UUID) ([]listing.Model, error)
	GetListingsPagedFunc       func(shopId uuid.UUID, page model.Page) (model.Paged[listing.Model], error)
	CreateShopFunc             func(characterId uint32, shopType shop.ShopType, title string, worldId world.Id, channelId channel.Id, mapId uint32, instanceId uuid.UUID, x int16, y int16, permitItemId uint32) (shop.Model, error)
	CreateShopAndEmitFunc      func(characterId uint32, shopType shop.ShopType, title string, worldId world.Id, channelId channel.Id, mapId uint32, instanceId uuid.UUID, x int16, y int16, permitItemId uint32) (shop.Model, error)
	OpenShopFunc               func(shopId uuid.UUID, characterId uint32) error
	EnterMaintenanceFunc       func(shopId uuid.UUID, characterId uint32) error
	ExitMaintenanceFunc        func(shopId uuid.UUID, characterId uint32) error
	CloseShopFunc              func(shopId uuid.UUID, characterId uint32, reason shop.CloseReason) error
	GetExpiredFunc             func() ([]shop.Model, error)
	AddListingFunc             func(shopId uuid.UUID, characterId uint32, itemId uint32, itemType byte, bundleSize uint16, bundleCount uint16, pricePerBundle uint32, itemSnapshot asset.AssetData, inventoryType byte, assetId uint32) (listing.Model, error)
	RemoveListingFunc          func(shopId uuid.UUID, characterId uint32, listingIndex uint16) (listing.Model, error)
	UpdateListingFunc          func(shopId uuid.UUID, listingIndex uint16, pricePerBundle uint32, bundleSize uint16, bundleCount uint16) error
	WithdrawMesoFunc           func(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32) error
	OrganizeListingsFunc       func(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32) error
	WithdrawMesoAndEmitFunc    func(shopId uuid.UUID, characterId uint32) error
	OrganizeListingsAndEmitFunc func(shopId uuid.UUID, characterId uint32) error
	EnterShopFunc              func(characterId uint32, shopId uuid.UUID, visitorName string) error
	ExitShopFunc               func(characterId uint32, shopId uuid.UUID) error
	EjectAllVisitorsFunc       func(shopId uuid.UUID) ([]uint32, error)
	GetVisitorsFunc            func(shopId uuid.UUID) ([]uint32, error)
	GetShopForCharacterFunc    func(characterId uint32) (uuid.UUID, error)
	PurchaseBundleFunc         func(buyerCharacterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16, worldId world.Id) (shop.PurchaseResult, error)
	SendMessageFunc            func(shopId uuid.UUID, characterId uint32, content string) error
	RetrieveFrederickFunc      func(characterId uint32, worldId world.Id) error
	OpenShopAndEmitFunc        func(shopId uuid.UUID, characterId uint32) error
	CloseShopAndEmitFunc       func(shopId uuid.UUID, characterId uint32, reason shop.CloseReason) error
	EnterMaintenanceAndEmitFunc func(shopId uuid.UUID, characterId uint32) error
	ExitMaintenanceAndEmitFunc  func(shopId uuid.UUID, characterId uint32) error
	EnterShopAndEmitFunc       func(characterId uint32, shopId uuid.UUID, visitorName string) error
	AddToBlacklistFunc         func(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32, name string, bannedCharacterId uint32) error
	RemoveFromBlacklistFunc    func(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32, name string) error
	GetBlacklistPagedFunc      func(shopId uuid.UUID, page model.Page) (model.Paged[string], error)
	GetVisitsPagedFunc         func(shopId uuid.UUID, page model.Page) (model.Paged[visit.Model], error)
	AddToBlacklistAndEmitFunc  func(shopId uuid.UUID, characterId uint32, name string, bannedCharacterId uint32) error
	RemoveFromBlacklistAndEmitFunc func(shopId uuid.UUID, characterId uint32, name string) error
	ExitShopAndEmitFunc        func(characterId uint32, shopId uuid.UUID) error
	AddListingAndEmitFunc      func(shopId uuid.UUID, characterId uint32, itemId uint32, itemType byte, bundleSize uint16, bundleCount uint16, pricePerBundle uint32, itemSnapshot asset.AssetData, inventoryType byte, assetId uint32) (listing.Model, error)
	RemoveListingAndEmitFunc   func(shopId uuid.UUID, characterId uint32, listingIndex uint16) (listing.Model, error)
	PurchaseBundleAndEmitFunc  func(buyerCharacterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16, worldId world.Id) (shop.PurchaseResult, error)
	SendMessageAndEmitFunc     func(shopId uuid.UUID, characterId uint32, content string) error
	RetrieveFrederickAndEmitFunc func(characterId uint32, worldId world.Id) error
}

func (m *ProcessorMock) WithTransaction(tx *gorm.DB) shop.Processor {
	if m.WithTransactionFunc != nil {
		return m.WithTransactionFunc(tx)
	}
	return m
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

func (m *ProcessorMock) GetByCharacterIdPaged(characterId uint32, page model.Page) (model.Paged[shop.Model], error) {
	if m.GetByCharacterIdPagedFunc != nil {
		return m.GetByCharacterIdPagedFunc(characterId, page)
	}
	return model.Paged[shop.Model]{Page: page}, nil
}

func (m *ProcessorMock) GetByField(worldId world.Id, channelId channel.Id, mapId uint32, instanceId uuid.UUID) ([]shop.Model, error) {
	if m.GetByFieldFunc != nil {
		return m.GetByFieldFunc(worldId, channelId, mapId, instanceId)
	}
	return []shop.Model{}, nil
}

func (m *ProcessorMock) GetByFieldPaged(worldId world.Id, channelId channel.Id, mapId uint32, instanceId uuid.UUID, page model.Page) (model.Paged[shop.Model], error) {
	if m.GetByFieldPagedFunc != nil {
		return m.GetByFieldPagedFunc(worldId, channelId, mapId, instanceId, page)
	}
	return model.Paged[shop.Model]{Page: page}, nil
}

func (m *ProcessorMock) GetAllOpenPaged(page model.Page) (model.Paged[shop.Model], error) {
	if m.GetAllOpenPagedFunc != nil {
		return m.GetAllOpenPagedFunc(page)
	}
	return model.Paged[shop.Model]{Page: page}, nil
}

func (m *ProcessorMock) GetListingCounts(shopIds []uuid.UUID) (map[uuid.UUID]int64, error) {
	if m.GetListingCountsFunc != nil {
		return m.GetListingCountsFunc(shopIds)
	}
	return make(map[uuid.UUID]int64), nil
}

func (m *ProcessorMock) SearchListingsByItemIdPaged(criteria shop.ListingSearchCriteria, page model.Page) (model.Paged[shop.ListingSearchResult], error) {
	if m.SearchListingsByItemIdPagedFunc != nil {
		return m.SearchListingsByItemIdPagedFunc(criteria, page)
	}
	return model.Paged[shop.ListingSearchResult]{Page: page}, nil
}

func (m *ProcessorMock) GetListings(shopId uuid.UUID) ([]listing.Model, error) {
	if m.GetListingsFunc != nil {
		return m.GetListingsFunc(shopId)
	}
	return []listing.Model{}, nil
}

func (m *ProcessorMock) GetListingsPaged(shopId uuid.UUID, page model.Page) (model.Paged[listing.Model], error) {
	if m.GetListingsPagedFunc != nil {
		return m.GetListingsPagedFunc(shopId, page)
	}
	return model.Paged[listing.Model]{Page: page}, nil
}

func (m *ProcessorMock) CreateShop(characterId uint32, shopType shop.ShopType, title string, worldId world.Id, channelId channel.Id, mapId uint32, instanceId uuid.UUID, x int16, y int16, permitItemId uint32) (shop.Model, error) {
	if m.CreateShopFunc != nil {
		return m.CreateShopFunc(characterId, shopType, title, worldId, channelId, mapId, instanceId, x, y, permitItemId)
	}
	return shop.Model{}, nil
}

func (m *ProcessorMock) CreateShopAndEmit(characterId uint32, shopType shop.ShopType, title string, worldId world.Id, channelId channel.Id, mapId uint32, instanceId uuid.UUID, x int16, y int16, permitItemId uint32) (shop.Model, error) {
	if m.CreateShopAndEmitFunc != nil {
		return m.CreateShopAndEmitFunc(characterId, shopType, title, worldId, channelId, mapId, instanceId, x, y, permitItemId)
	}
	return shop.Model{}, nil
}

func (m *ProcessorMock) OpenShop(_ *message.Buffer) func(shopId uuid.UUID, characterId uint32) error {
	return func(shopId uuid.UUID, characterId uint32) error {
		if m.OpenShopFunc != nil {
			return m.OpenShopFunc(shopId, characterId)
		}
		return nil
	}
}

func (m *ProcessorMock) EnterMaintenance(_ *message.Buffer) func(shopId uuid.UUID, characterId uint32) error {
	return func(shopId uuid.UUID, characterId uint32) error {
		if m.EnterMaintenanceFunc != nil {
			return m.EnterMaintenanceFunc(shopId, characterId)
		}
		return nil
	}
}

func (m *ProcessorMock) ExitMaintenance(_ *message.Buffer) func(shopId uuid.UUID, characterId uint32) error {
	return func(shopId uuid.UUID, characterId uint32) error {
		if m.ExitMaintenanceFunc != nil {
			return m.ExitMaintenanceFunc(shopId, characterId)
		}
		return nil
	}
}

func (m *ProcessorMock) CloseShop(_ *message.Buffer) func(shopId uuid.UUID, characterId uint32, reason shop.CloseReason) error {
	return func(shopId uuid.UUID, characterId uint32, reason shop.CloseReason) error {
		if m.CloseShopFunc != nil {
			return m.CloseShopFunc(shopId, characterId, reason)
		}
		return nil
	}
}

func (m *ProcessorMock) GetExpired() ([]shop.Model, error) {
	if m.GetExpiredFunc != nil {
		return m.GetExpiredFunc()
	}
	return []shop.Model{}, nil
}

func (m *ProcessorMock) AddListing(_ *message.Buffer) func(shopId uuid.UUID, characterId uint32, itemId uint32, itemType byte, bundleSize uint16, bundleCount uint16, pricePerBundle uint32, itemSnapshot asset.AssetData, inventoryType byte, assetId uint32) (listing.Model, error) {
	return func(shopId uuid.UUID, characterId uint32, itemId uint32, itemType byte, bundleSize uint16, bundleCount uint16, pricePerBundle uint32, itemSnapshot asset.AssetData, inventoryType byte, assetId uint32) (listing.Model, error) {
		if m.AddListingFunc != nil {
			return m.AddListingFunc(shopId, characterId, itemId, itemType, bundleSize, bundleCount, pricePerBundle, itemSnapshot, inventoryType, assetId)
		}
		return listing.Model{}, nil
	}
}

func (m *ProcessorMock) RemoveListing(_ *message.Buffer) func(shopId uuid.UUID, characterId uint32, listingIndex uint16) (listing.Model, error) {
	return func(shopId uuid.UUID, characterId uint32, listingIndex uint16) (listing.Model, error) {
		if m.RemoveListingFunc != nil {
			return m.RemoveListingFunc(shopId, characterId, listingIndex)
		}
		return listing.Model{}, nil
	}
}

func (m *ProcessorMock) UpdateListing(shopId uuid.UUID, listingIndex uint16, pricePerBundle uint32, bundleSize uint16, bundleCount uint16) error {
	if m.UpdateListingFunc != nil {
		return m.UpdateListingFunc(shopId, listingIndex, pricePerBundle, bundleSize, bundleCount)
	}
	return nil
}

func (m *ProcessorMock) WithdrawMeso(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32) error {
	if m.WithdrawMesoFunc != nil {
		return m.WithdrawMesoFunc(mb)
	}
	return func(uuid.UUID, uint32) error { return nil }
}

func (m *ProcessorMock) OrganizeListings(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32) error {
	if m.OrganizeListingsFunc != nil {
		return m.OrganizeListingsFunc(mb)
	}
	return func(uuid.UUID, uint32) error { return nil }
}

func (m *ProcessorMock) WithdrawMesoAndEmit(shopId uuid.UUID, characterId uint32) error {
	if m.WithdrawMesoAndEmitFunc != nil {
		return m.WithdrawMesoAndEmitFunc(shopId, characterId)
	}
	return nil
}

func (m *ProcessorMock) OrganizeListingsAndEmit(shopId uuid.UUID, characterId uint32) error {
	if m.OrganizeListingsAndEmitFunc != nil {
		return m.OrganizeListingsAndEmitFunc(shopId, characterId)
	}
	return nil
}

func (m *ProcessorMock) EnterShop(_ *message.Buffer) func(characterId uint32, shopId uuid.UUID, visitorName string) error {
	return func(characterId uint32, shopId uuid.UUID, visitorName string) error {
		if m.EnterShopFunc != nil {
			return m.EnterShopFunc(characterId, shopId, visitorName)
		}
		return nil
	}
}

func (m *ProcessorMock) ExitShop(_ *message.Buffer) func(characterId uint32, shopId uuid.UUID) error {
	return func(characterId uint32, shopId uuid.UUID) error {
		if m.ExitShopFunc != nil {
			return m.ExitShopFunc(characterId, shopId)
		}
		return nil
	}
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

func (m *ProcessorMock) GetShopForCharacter(characterId uint32) (uuid.UUID, error) {
	if m.GetShopForCharacterFunc != nil {
		return m.GetShopForCharacterFunc(characterId)
	}
	return uuid.Nil, nil
}

func (m *ProcessorMock) PurchaseBundle(_ *message.Buffer) func(buyerCharacterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16, worldId world.Id) (shop.PurchaseResult, error) {
	return func(buyerCharacterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16, worldId world.Id) (shop.PurchaseResult, error) {
		if m.PurchaseBundleFunc != nil {
			return m.PurchaseBundleFunc(buyerCharacterId, shopId, listingIndex, bundleCount, worldId)
		}
		return shop.PurchaseResult{}, nil
	}
}

func (m *ProcessorMock) SendMessage(_ *message.Buffer) func(shopId uuid.UUID, characterId uint32, content string) error {
	return func(shopId uuid.UUID, characterId uint32, content string) error {
		if m.SendMessageFunc != nil {
			return m.SendMessageFunc(shopId, characterId, content)
		}
		return nil
	}
}

func (m *ProcessorMock) RetrieveFrederick(_ *message.Buffer) func(characterId uint32, worldId world.Id) error {
	return func(characterId uint32, worldId world.Id) error {
		if m.RetrieveFrederickFunc != nil {
			return m.RetrieveFrederickFunc(characterId, worldId)
		}
		return nil
	}
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

func (m *ProcessorMock) EnterShopAndEmit(characterId uint32, shopId uuid.UUID, visitorName string) error {
	if m.EnterShopAndEmitFunc != nil {
		return m.EnterShopAndEmitFunc(characterId, shopId, visitorName)
	}
	return nil
}

func (m *ProcessorMock) AddToBlacklist(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32, name string, bannedCharacterId uint32) error {
	if m.AddToBlacklistFunc != nil {
		return m.AddToBlacklistFunc(mb)
	}
	return func(uuid.UUID, uint32, string, uint32) error { return nil }
}

func (m *ProcessorMock) RemoveFromBlacklist(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32, name string) error {
	if m.RemoveFromBlacklistFunc != nil {
		return m.RemoveFromBlacklistFunc(mb)
	}
	return func(uuid.UUID, uint32, string) error { return nil }
}

func (m *ProcessorMock) GetBlacklistPaged(shopId uuid.UUID, page model.Page) (model.Paged[string], error) {
	if m.GetBlacklistPagedFunc != nil {
		return m.GetBlacklistPagedFunc(shopId, page)
	}
	return model.Paged[string]{Items: []string{}, Page: page}, nil
}

func (m *ProcessorMock) GetVisitsPaged(shopId uuid.UUID, page model.Page) (model.Paged[visit.Model], error) {
	if m.GetVisitsPagedFunc != nil {
		return m.GetVisitsPagedFunc(shopId, page)
	}
	return model.Paged[visit.Model]{Items: []visit.Model{}, Page: page}, nil
}

func (m *ProcessorMock) AddToBlacklistAndEmit(shopId uuid.UUID, characterId uint32, name string, bannedCharacterId uint32) error {
	if m.AddToBlacklistAndEmitFunc != nil {
		return m.AddToBlacklistAndEmitFunc(shopId, characterId, name, bannedCharacterId)
	}
	return nil
}

func (m *ProcessorMock) RemoveFromBlacklistAndEmit(shopId uuid.UUID, characterId uint32, name string) error {
	if m.RemoveFromBlacklistAndEmitFunc != nil {
		return m.RemoveFromBlacklistAndEmitFunc(shopId, characterId, name)
	}
	return nil
}

func (m *ProcessorMock) ExitShopAndEmit(characterId uint32, shopId uuid.UUID) error {
	if m.ExitShopAndEmitFunc != nil {
		return m.ExitShopAndEmitFunc(characterId, shopId)
	}
	return nil
}

func (m *ProcessorMock) AddListingAndEmit(shopId uuid.UUID, characterId uint32, itemId uint32, itemType byte, bundleSize uint16, bundleCount uint16, pricePerBundle uint32, itemSnapshot asset.AssetData, inventoryType byte, assetId uint32) (listing.Model, error) {
	if m.AddListingAndEmitFunc != nil {
		return m.AddListingAndEmitFunc(shopId, characterId, itemId, itemType, bundleSize, bundleCount, pricePerBundle, itemSnapshot, inventoryType, assetId)
	}
	return listing.Model{}, nil
}

func (m *ProcessorMock) RemoveListingAndEmit(shopId uuid.UUID, characterId uint32, listingIndex uint16) (listing.Model, error) {
	if m.RemoveListingAndEmitFunc != nil {
		return m.RemoveListingAndEmitFunc(shopId, characterId, listingIndex)
	}
	return listing.Model{}, nil
}

func (m *ProcessorMock) PurchaseBundleAndEmit(buyerCharacterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16, worldId world.Id) (shop.PurchaseResult, error) {
	if m.PurchaseBundleAndEmitFunc != nil {
		return m.PurchaseBundleAndEmitFunc(buyerCharacterId, shopId, listingIndex, bundleCount, worldId)
	}
	return shop.PurchaseResult{}, nil
}

func (m *ProcessorMock) SendMessageAndEmit(shopId uuid.UUID, characterId uint32, content string) error {
	if m.SendMessageAndEmitFunc != nil {
		return m.SendMessageAndEmitFunc(shopId, characterId, content)
	}
	return nil
}

func (m *ProcessorMock) RetrieveFrederickAndEmit(characterId uint32, worldId world.Id) error {
	if m.RetrieveFrederickAndEmitFunc != nil {
		return m.RetrieveFrederickAndEmitFunc(characterId, worldId)
	}
	return nil
}
