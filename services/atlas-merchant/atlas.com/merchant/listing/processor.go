package listing

import (
	"atlas-merchant/kafka/message/asset"
	"time"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Processor interface {
	GetByShopId(shopId uuid.UUID) ([]Model, error)
	GetByShopIdAndDisplayOrder(shopId uuid.UUID, displayOrder uint16) (Model, error)
	CountByShopId(shopId uuid.UUID) (int64, error)
	CountByShopIds(shopIds []uuid.UUID) (map[uuid.UUID]int64, error)
	Create(shopId uuid.UUID, tenantId uuid.UUID, itemId uint32, itemType byte, bundleSize uint16, bundleCount uint16, pricePerBundle uint32, itemSnapshot asset.AssetData, displayOrder uint16) (Model, error)
	Delete(id uuid.UUID) error
	DeleteByShopId(shopId uuid.UUID) error
	UpdateBundles(id uuid.UUID, bundlesRemaining uint16, quantity uint16, expectedVersion uint32) (int64, error)
	DecrementDisplayOrderAfter(shopId uuid.UUID, afterOrder uint16) error
	UpdateFields(id uuid.UUID, pricePerBundle uint32, bundleSize uint16, bundleCount uint16) error
}

type ProcessorImpl struct {
	db *gorm.DB
}

func NewProcessor(db *gorm.DB) Processor {
	return &ProcessorImpl{db: db}
}

func (p *ProcessorImpl) GetByShopId(shopId uuid.UUID) ([]Model, error) {
	return model.SliceMap(Make)(getByShopId(shopId)(p.db))(model.ParallelMap())()
}

func (p *ProcessorImpl) GetByShopIdAndDisplayOrder(shopId uuid.UUID, displayOrder uint16) (Model, error) {
	e, err := getByShopIdAndDisplayOrder(shopId, displayOrder)(p.db)()
	if err != nil {
		return Model{}, err
	}
	return Make(e)
}

func (p *ProcessorImpl) CountByShopId(shopId uuid.UUID) (int64, error) {
	return countByShopId(shopId)(p.db)()
}

func (p *ProcessorImpl) CountByShopIds(shopIds []uuid.UUID) (map[uuid.UUID]int64, error) {
	return countByShopIds(shopIds)(p.db)()
}

func (p *ProcessorImpl) Create(shopId uuid.UUID, tenantId uuid.UUID, itemId uint32, itemType byte, bundleSize uint16, bundleCount uint16, pricePerBundle uint32, itemSnapshot asset.AssetData, displayOrder uint16) (Model, error) {
	entity := &Entity{
		Id:               uuid.New(),
		TenantId:         tenantId,
		ShopId:           shopId,
		ItemId:           itemId,
		ItemType:         itemType,
		Quantity:         bundleSize * bundleCount,
		BundleSize:       bundleSize,
		BundlesRemaining: bundleCount,
		PricePerBundle:   pricePerBundle,
		ItemSnapshot:     itemSnapshot,
		DisplayOrder:     displayOrder,
		Version:          1,
		ListedAt:         time.Now(),
	}

	le, err := createListing(entity)(p.db)()
	if err != nil {
		return Model{}, err
	}
	return Make(le)
}

func (p *ProcessorImpl) Delete(id uuid.UUID) error {
	_, err := deleteListing(id)(p.db)()
	return err
}

func (p *ProcessorImpl) DeleteByShopId(shopId uuid.UUID) error {
	_, err := deleteByShopId(shopId)(p.db)()
	return err
}

func (p *ProcessorImpl) UpdateBundles(id uuid.UUID, bundlesRemaining uint16, quantity uint16, expectedVersion uint32) (int64, error) {
	return updateBundles(id, bundlesRemaining, quantity, expectedVersion)(p.db)()
}

func (p *ProcessorImpl) DecrementDisplayOrderAfter(shopId uuid.UUID, afterOrder uint16) error {
	_, err := decrementDisplayOrderAfter(shopId, afterOrder)(p.db)()
	return err
}

func (p *ProcessorImpl) UpdateFields(id uuid.UUID, pricePerBundle uint32, bundleSize uint16, bundleCount uint16) error {
	_, err := updateListingFields(id, pricePerBundle, bundleSize, bundleCount)(p.db)()
	return err
}
