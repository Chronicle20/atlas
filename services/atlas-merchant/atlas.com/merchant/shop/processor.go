package shop

import (
	"atlas-merchant/frederick"
	kafkaProducer "atlas-merchant/kafka/producer"
	"atlas-merchant/listing"
	"atlas-merchant/visitor"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	database "github.com/Chronicle20/atlas-database"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const MaxListings = 16
const MaxVisitors = 3

type Processor interface {
	GetById(id uuid.UUID) (Model, error)
	ByIdProvider(id uuid.UUID) model.Provider[Model]
	GetByCharacterId(characterId uint32) ([]Model, error)
	GetByMapId(mapId uint32) ([]Model, error)
	GetAllOpen() ([]Model, error)
	GetListingCounts(shopIds []uuid.UUID) (map[uuid.UUID]int64, error)
	SearchListingsByItemId(itemId uint32) ([]ListingSearchResult, error)
	GetListings(shopId uuid.UUID) ([]listing.Model, error)
	CreateShop(characterId uint32, shopType ShopType, title string, mapId uint32, x int16, y int16, permitItemId uint32) (Model, error)
	OpenShop(shopId uuid.UUID) error
	EnterMaintenance(shopId uuid.UUID) error
	ExitMaintenance(shopId uuid.UUID) (bool, error)
	CloseShop(shopId uuid.UUID, reason CloseReason) error
	GetExpired() ([]Model, error)
	AddListing(shopId uuid.UUID, itemId uint32, itemType byte, bundleSize uint16, bundleCount uint16, pricePerBundle uint32, itemSnapshot json.RawMessage, flag uint16) (listing.Model, error)
	RemoveListing(shopId uuid.UUID, listingIndex uint16) (listing.Model, error)
	UpdateListing(shopId uuid.UUID, listingIndex uint16, pricePerBundle uint32, bundleSize uint16, bundleCount uint16) error
	EnterShop(characterId uint32, shopId uuid.UUID) error
	ExitShop(characterId uint32, shopId uuid.UUID) error
	EjectAllVisitors(shopId uuid.UUID) ([]uint32, error)
	GetVisitors(shopId uuid.UUID) ([]uint32, error)
	PurchaseBundle(buyerCharacterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16) (PurchaseResult, error)
	OpenShopAndEmit(shopId uuid.UUID, characterId uint32) error
	CloseShopAndEmit(shopId uuid.UUID, characterId uint32, reason CloseReason) error
	EnterMaintenanceAndEmit(shopId uuid.UUID, characterId uint32) error
	ExitMaintenanceAndEmit(shopId uuid.UUID, characterId uint32) error
	EnterShopAndEmit(characterId uint32, shopId uuid.UUID) error
	ExitShopAndEmit(characterId uint32, shopId uuid.UUID) error
	AddListingAndEmit(shopId uuid.UUID, characterId uint32, itemId uint32, itemType byte, bundleSize uint16, bundleCount uint16, pricePerBundle uint32, itemSnapshot json.RawMessage, flag uint16, inventoryType byte, assetId uint32) (listing.Model, error)
	PurchaseBundleAndEmit(buyerCharacterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16, worldId world.Id) (PurchaseResult, error)
}

type PurchaseResult struct {
	ListingId        uuid.UUID
	ItemId           uint32
	ItemType         byte
	ItemSnapshot     json.RawMessage
	BundleSize       uint16
	BundlesPurchased uint16
	BundlesRemaining uint16
	TotalCost        int64
	Fee              int64
	NetAmount        int64
	ShopOwnerId      uint32
	ShopType         ShopType
	ShopClosed       bool
}

type ListingSearchResult struct {
	Listing listing.Model
	ShopId  uuid.UUID
	Title   string
	MapId   uint32
}

var ErrNotFound = errors.New("not found")
var ErrInvalidTransition = errors.New("invalid state transition")
var ErrShopLimitReached = errors.New("active shop limit reached")
var ErrListingLimitReached = errors.New("listing limit reached")
var ErrInsufficientBundles = errors.New("insufficient bundles")
var ErrVersionConflict = errors.New("version conflict")
var ErrNoListings = errors.New("shop has no listings")
var ErrShopFull = errors.New("shop is full")
var ErrFrederickPending = errors.New("items or mesos pending at Frederick")

type ProcessorImpl struct {
	l        logrus.FieldLogger
	ctx      context.Context
	db       *gorm.DB
	t        tenant.Model
	producer kafkaProducer.Provider
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:        l,
		ctx:      ctx,
		db:       db,
		t:        tenant.MustFromContext(ctx),
		producer: kafkaProducer.ProviderImpl(l)(ctx),
	}
}

func (p *ProcessorImpl) GetById(id uuid.UUID) (Model, error) {
	return p.ByIdProvider(id)()
}

func (p *ProcessorImpl) ByIdProvider(id uuid.UUID) model.Provider[Model] {
	return model.Map(Make)(getById(id)(p.db.WithContext(p.ctx)))
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	return model.SliceMap(Make)(getByCharacterId(characterId)(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}

func (p *ProcessorImpl) GetByMapId(mapId uint32) ([]Model, error) {
	return model.SliceMap(Make)(getByMapId(mapId)(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}

func (p *ProcessorImpl) GetAllOpen() ([]Model, error) {
	return model.SliceMap(Make)(getAllOpen()(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}

func (p *ProcessorImpl) GetListingCounts(shopIds []uuid.UUID) (map[uuid.UUID]int64, error) {
	return listing.CountByShopIds(shopIds)(p.db.WithContext(p.ctx))()
}

func (p *ProcessorImpl) SearchListingsByItemId(itemId uint32) ([]ListingSearchResult, error) {
	return searchListingsByItemId(itemId)(p.db.WithContext(p.ctx))()
}

func (p *ProcessorImpl) GetListings(shopId uuid.UUID) ([]listing.Model, error) {
	return model.SliceMap(listing.Make)(listing.GetByShopId(shopId)(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}

func (p *ProcessorImpl) GetExpired() ([]Model, error) {
	return model.SliceMap(Make)(getExpired()(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}

func (p *ProcessorImpl) CreateShop(characterId uint32, shopType ShopType, title string, mapId uint32, x int16, y int16, permitItemId uint32) (Model, error) {
	// Validate Free Market room.
	if !IsFreemarketRoom(mapId) {
		return Model{}, ErrNotFreemarketRoom
	}

	// Validate portal proximity.
	if IsNearPortal(p.l, p.ctx, mapId, x, y) {
		return Model{}, ErrTooCloseToPortal
	}

	// Validate shop-to-shop proximity.
	shopProvider := func() ([]Model, error) {
		return p.GetByMapId(mapId)
	}
	if IsNearExistingShop(mapId, x, y, shopProvider) {
		return Model{}, ErrTooCloseToShop
	}

	// Check active shop limit — one per type per character.
	_, err := getActiveByCharacterIdAndType(characterId, shopType)(p.db.WithContext(p.ctx))()
	if err == nil {
		return Model{}, ErrShopLimitReached
	}
	if !errors.Is(err, ErrNotFound) {
		return Model{}, err
	}

	// Block placement if character has items/mesos waiting at Frederick.
	hasPending, err := frederick.HasItemsOrMesos(characterId)(p.db.WithContext(p.ctx))()
	if err != nil {
		return Model{}, err
	}
	if hasPending {
		return Model{}, ErrFrederickPending
	}

	now := time.Now()
	id := uuid.New()

	entity := &Entity{
		Id:           id,
		TenantId:     p.t.Id(),
		TenantRegion: p.t.Region(),
		TenantMajor:  p.t.MajorVersion(),
		TenantMinor:  p.t.MinorVersion(),
		CharacterId:  characterId,
		ShopType:     byte(shopType),
		State:        byte(Draft),
		Title:        title,
		MapId:        mapId,
		X:            x,
		Y:            y,
		PermitItemId: permitItemId,
		CloseReason:  byte(CloseReasonNone),
		MesoBalance:  0,
	}
	entity.CreatedAt = now

	if shopType == HiredMerchant {
		expires := now.Add(24 * time.Hour)
		entity.ExpiresAt = &expires
	}

	e, err := create(entity)(p.db.WithContext(p.ctx))()
	if err != nil {
		return Model{}, err
	}

	m, err := Make(e)
	if err != nil {
		return Model{}, err
	}

	// Register in Redis.
	r := GetRegistry()
	if r != nil {
		_ = r.activeShops.Put(p.ctx, p.t, characterId, ActiveShopEntry{
			ShopId:   id,
			ShopType: shopType,
			MapId:    mapId,
		})
	}

	p.l.Infof("Shop [%s] created for character [%d], type [%d].", id, characterId, shopType)
	return m, nil
}

func (p *ProcessorImpl) OpenShop(shopId uuid.UUID) error {
	var mapId uint32
	err := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		e, err := getById(shopId)(tx)()
		if err != nil {
			return err
		}

		if State(e.State) != Draft {
			return fmt.Errorf("%w: cannot open shop in state %d", ErrInvalidTransition, e.State)
		}

		// Verify at least one listing exists.
		count, err := listing.CountByShopId(shopId)(tx)()
		if err != nil {
			return err
		}
		if count == 0 {
			return ErrNoListings
		}

		e.State = byte(Open)
		_, err = update(&e)(tx)()
		if err != nil {
			return err
		}

		mapId = e.MapId
		return nil
	})
	if err != nil {
		return err
	}

	// Add to map placement index.
	p.addToMapIndex(mapId, shopId)
	return nil
}

func (p *ProcessorImpl) EnterMaintenance(shopId uuid.UUID) error {
	err := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		e, err := getById(shopId)(tx)()
		if err != nil {
			return err
		}

		if State(e.State) != Open {
			return fmt.Errorf("%w: cannot enter maintenance in state %d", ErrInvalidTransition, e.State)
		}

		e.State = byte(Maintenance)
		_, err = update(&e)(tx)()
		return err
	})
	if err != nil {
		return err
	}

	ejected, _ := p.EjectAllVisitors(shopId)
	if len(ejected) > 0 {
		p.l.Infof("Shop [%s] entered maintenance, ejected %d visitors.", shopId, len(ejected))
	}
	return nil
}

func (p *ProcessorImpl) ExitMaintenance(shopId uuid.UUID) (bool, error) {
	var closed bool
	var characterId uint32
	var mapId uint32
	err := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		e, err := getById(shopId)(tx)()
		if err != nil {
			return err
		}

		if State(e.State) != Maintenance {
			return fmt.Errorf("%w: cannot exit maintenance in state %d", ErrInvalidTransition, e.State)
		}

		// If no listings remain, close the shop.
		count, err := listing.CountByShopId(shopId)(tx)()
		if err != nil {
			return err
		}

		if count == 0 {
			now := time.Now()
			e.State = byte(Closed)
			e.ClosedAt = &now
			e.CloseReason = byte(CloseReasonEmpty)
		} else {
			e.State = byte(Open)
		}

		_, err = update(&e)(tx)()
		if err != nil {
			return err
		}

		closed = State(e.State) == Closed
		characterId = e.CharacterId
		mapId = e.MapId
		return nil
	})
	if err != nil {
		return false, err
	}

	if closed {
		p.removeFromRegistry(characterId)
		p.removeFromMapIndex(mapId, shopId)
	}
	return closed, nil
}

func (p *ProcessorImpl) CloseShop(shopId uuid.UUID, reason CloseReason) error {
	var characterId uint32
	var mapId uint32
	var shopType ShopType
	var mesoBalance uint32
	err := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		e, err := getById(shopId)(tx)()
		if err != nil {
			return err
		}

		currentState := State(e.State)
		if currentState != Open && currentState != Maintenance && currentState != Draft {
			return fmt.Errorf("%w: cannot close shop in state %d", ErrInvalidTransition, e.State)
		}

		now := time.Now()
		e.State = byte(Closed)
		e.ClosedAt = &now
		e.CloseReason = byte(reason)

		_, err = update(&e)(tx)()
		if err != nil {
			return err
		}

		characterId = e.CharacterId
		mapId = e.MapId
		shopType = ShopType(e.ShopType)
		mesoBalance = e.MesoBalance
		return nil
	})
	if err != nil {
		return err
	}

	p.removeFromRegistry(characterId)
	p.removeFromMapIndex(mapId, shopId)
	p.EjectAllVisitors(shopId)

	// For hired merchants, move unsold items and mesos to Frederick.
	if shopType == HiredMerchant {
		p.storeToFrederick(shopId, characterId, mesoBalance)
	}

	p.l.Infof("Shop [%s] closed, reason [%d].", shopId, reason)
	return nil
}

func (p *ProcessorImpl) storeToFrederick(shopId uuid.UUID, characterId uint32, mesoBalance uint32) {
	listings, err := p.GetListings(shopId)
	if err != nil {
		p.l.WithError(err).Errorf("Error retrieving listings for Frederick storage, shop [%s].", shopId)
		return
	}

	fp := frederick.NewProcessor(p.l, p.ctx, p.db)

	if len(listings) > 0 {
		items := make([]frederick.StoredItem, 0, len(listings))
		for _, l := range listings {
			items = append(items, frederick.StoredItem{
				ItemId:       l.ItemId(),
				ItemType:     l.ItemType(),
				Quantity:     l.Quantity(),
				ItemSnapshot: l.ItemSnapshot(),
			})
		}

		if err := fp.StoreItems(characterId, items); err != nil {
			p.l.WithError(err).Errorf("Error storing items to Frederick for character [%d].", characterId)
		}
	}

	if mesoBalance > 0 {
		if err := fp.StoreMesos(characterId, mesoBalance); err != nil {
			p.l.WithError(err).Errorf("Error storing mesos to Frederick for character [%d].", characterId)
		}
	}

	// Create notification for Frederick retrieval reminders.
	if len(listings) > 0 || mesoBalance > 0 {
		if err := fp.CreateNotification(characterId); err != nil {
			p.l.WithError(err).Errorf("Error creating Frederick notification for character [%d].", characterId)
		}
	}
}

func (p *ProcessorImpl) AddListing(shopId uuid.UUID, itemId uint32, itemType byte, bundleSize uint16, bundleCount uint16, pricePerBundle uint32, itemSnapshot json.RawMessage, flag uint16) (listing.Model, error) {
	if pricePerBundle == 0 {
		return listing.Model{}, errors.New("pricePerBundle must be at least 1")
	}
	if bundleSize == 0 {
		return listing.Model{}, errors.New("bundleSize must be at least 1")
	}
	if bundleCount == 0 {
		return listing.Model{}, errors.New("bundleCount must be at least 1")
	}

	// Trade restriction validation.
	if err := IsListableItem(itemId, flag); err != nil {
		return listing.Model{}, err
	}

	var result listing.Model
	err := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		e, err := getById(shopId)(tx)()
		if err != nil {
			return err
		}

		if State(e.State) != Draft && State(e.State) != Maintenance {
			return fmt.Errorf("%w: cannot add listing in state %d", ErrInvalidTransition, e.State)
		}

		count, err := listing.CountByShopId(shopId)(tx)()
		if err != nil {
			return err
		}
		if count >= MaxListings {
			return ErrListingLimitReached
		}

		entity := &listing.Entity{
			Id:               uuid.New(),
			TenantId:         p.t.Id(),
			ShopId:           shopId,
			ItemId:           itemId,
			ItemType:         itemType,
			Quantity:         bundleSize * bundleCount,
			BundleSize:       bundleSize,
			BundlesRemaining: bundleCount,
			PricePerBundle:   pricePerBundle,
			ItemSnapshot:     itemSnapshot,
			DisplayOrder:     uint16(count),
			Version:          1,
			ListedAt:         time.Now(),
		}

		le, err := listing.CreateListing(entity)(tx)()
		if err != nil {
			return err
		}

		result, err = listing.Make(le)
		return err
	})
	return result, err
}

func (p *ProcessorImpl) RemoveListing(shopId uuid.UUID, listingIndex uint16) (listing.Model, error) {
	var result listing.Model
	err := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		e, err := getById(shopId)(tx)()
		if err != nil {
			return err
		}

		if State(e.State) != Draft && State(e.State) != Maintenance {
			return fmt.Errorf("%w: cannot remove listing in state %d", ErrInvalidTransition, e.State)
		}

		le, err := listing.GetByShopIdAndDisplayOrder(shopId, listingIndex)(tx)()
		if err != nil {
			return err
		}

		result, err = listing.Make(le)
		if err != nil {
			return err
		}

		_, err = listing.DeleteListing(le.Id)(tx)()
		if err != nil {
			return err
		}

		_, err = listing.DecrementDisplayOrderAfter(shopId, listingIndex)(tx)()
		return err
	})
	return result, err
}

func (p *ProcessorImpl) UpdateListing(shopId uuid.UUID, listingIndex uint16, pricePerBundle uint32, bundleSize uint16, bundleCount uint16) error {
	if pricePerBundle == 0 {
		return errors.New("pricePerBundle must be at least 1")
	}
	if bundleSize == 0 {
		return errors.New("bundleSize must be at least 1")
	}
	if bundleCount == 0 {
		return errors.New("bundleCount must be at least 1")
	}

	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		e, err := getById(shopId)(tx)()
		if err != nil {
			return err
		}

		if State(e.State) != Draft && State(e.State) != Maintenance {
			return fmt.Errorf("%w: cannot update listing in state %d", ErrInvalidTransition, e.State)
		}

		le, err := listing.GetByShopIdAndDisplayOrder(shopId, listingIndex)(tx)()
		if err != nil {
			return err
		}

		_, err = listing.UpdateListingFields(le.Id, pricePerBundle, bundleSize, bundleCount)(tx)()
		return err
	})
}

func (p *ProcessorImpl) PurchaseBundle(buyerCharacterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16) (PurchaseResult, error) {
	if bundleCount == 0 {
		return PurchaseResult{}, errors.New("bundleCount must be at least 1")
	}

	var result PurchaseResult
	var mapId uint32
	err := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		e, err := getById(shopId)(tx)()
		if err != nil {
			return err
		}

		if State(e.State) != Open {
			return fmt.Errorf("%w: shop not open for purchase", ErrInvalidTransition)
		}

		result.ShopOwnerId = e.CharacterId
		result.ShopType = ShopType(e.ShopType)
		mapId = e.MapId

		le, err := listing.GetByShopIdAndDisplayOrder(shopId, listingIndex)(tx)()
		if err != nil {
			return err
		}

		if le.BundlesRemaining < bundleCount {
			return ErrInsufficientBundles
		}

		totalCost := int64(bundleCount) * int64(le.PricePerBundle)
		fee := GetFee(totalCost)

		result.ListingId = le.Id
		result.ItemId = le.ItemId
		result.ItemType = le.ItemType
		result.ItemSnapshot = le.ItemSnapshot
		result.BundleSize = le.BundleSize
		result.BundlesPurchased = bundleCount
		result.TotalCost = totalCost
		result.Fee = fee
		result.NetAmount = totalCost - fee

		newBundlesRemaining := le.BundlesRemaining - bundleCount
		newQuantity := le.BundleSize * newBundlesRemaining

		// Optimistic lock: only succeeds if version matches.
		rowsAffected, err := listing.UpdateBundles(le.Id, newBundlesRemaining, newQuantity, le.Version)(tx)()
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return ErrVersionConflict
		}

		result.BundlesRemaining = newBundlesRemaining

		// If listing sold out, remove it and collapse display order.
		if newBundlesRemaining == 0 {
			_, err = listing.DeleteListing(le.Id)(tx)()
			if err != nil {
				return err
			}

			_, err = listing.DecrementDisplayOrderAfter(shopId, listingIndex)(tx)()
			if err != nil {
				return err
			}

			// Check if all listings are gone → close shop (sold out).
			count, err := listing.CountByShopId(shopId)(tx)()
			if err != nil {
				return err
			}

			if count == 0 {
				now := time.Now()
				e.State = byte(Closed)
				e.ClosedAt = &now
				e.CloseReason = byte(CloseReasonSoldOut)
				result.ShopClosed = true
			}
		}

		// Accumulate meso balance for hired merchants.
		if ShopType(e.ShopType) == HiredMerchant {
			e.MesoBalance += uint32(totalCost - fee)
		}

		// Persist entity changes if needed (meso balance or shop closure).
		if ShopType(e.ShopType) == HiredMerchant || result.ShopClosed {
			_, err = update(&e)(tx)()
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return PurchaseResult{}, err
	}

	if result.ShopClosed {
		p.removeFromRegistry(result.ShopOwnerId)
		p.removeFromMapIndex(mapId, shopId)
		p.EjectAllVisitors(shopId)
		p.l.Infof("Shop [%s] sold out and closed.", shopId)
	}

	return result, nil
}

func (p *ProcessorImpl) EnterShop(characterId uint32, shopId uuid.UUID) error {
	e, err := getById(shopId)(p.db.WithContext(p.ctx))()
	if err != nil {
		return err
	}

	if State(e.State) != Open {
		return fmt.Errorf("%w: shop not open", ErrInvalidTransition)
	}

	vr := visitor.GetRegistry()
	if vr == nil {
		return errors.New("visitor registry not initialized")
	}

	count, err := vr.GetVisitorCount(p.ctx, p.t, shopId)
	if err != nil {
		return err
	}

	if count >= MaxVisitors {
		return ErrShopFull
	}

	return vr.AddVisitor(p.ctx, p.t, shopId, characterId)
}

func (p *ProcessorImpl) ExitShop(characterId uint32, shopId uuid.UUID) error {
	vr := visitor.GetRegistry()
	if vr == nil {
		return errors.New("visitor registry not initialized")
	}
	return vr.RemoveVisitor(p.ctx, p.t, shopId, characterId)
}

func (p *ProcessorImpl) EjectAllVisitors(shopId uuid.UUID) ([]uint32, error) {
	vr := visitor.GetRegistry()
	if vr == nil {
		return nil, nil
	}
	return vr.RemoveAllVisitors(p.ctx, p.t, shopId)
}

func (p *ProcessorImpl) GetVisitors(shopId uuid.UUID) ([]uint32, error) {
	vr := visitor.GetRegistry()
	if vr == nil {
		return nil, nil
	}
	return vr.GetVisitors(p.ctx, p.t, shopId)
}

func (p *ProcessorImpl) removeFromRegistry(characterId uint32) {
	r := GetRegistry()
	if r != nil {
		_ = r.activeShops.Remove(p.ctx, p.t, characterId)
	}
}

func (p *ProcessorImpl) addToMapIndex(mapId uint32, shopId uuid.UUID) {
	r := GetRegistry()
	if r != nil {
		mapKey := strconv.FormatUint(uint64(mapId), 10)
		_ = r.mapPlacement.Add(p.ctx, p.t, mapKey, shopId.String())
	}
}

func (p *ProcessorImpl) removeFromMapIndex(mapId uint32, shopId uuid.UUID) {
	r := GetRegistry()
	if r != nil {
		mapKey := strconv.FormatUint(uint64(mapId), 10)
		_ = r.mapPlacement.Remove(p.ctx, p.t, mapKey, shopId.String())
	}
}

// GetFee calculates the sale fee based on the total meso amount.
func GetFee(meso int64) int64 {
	var fee int64
	if meso >= 100000000 {
		fee = (meso * 6) / 100
	} else if meso >= 25000000 {
		fee = (meso * 5) / 100
	} else if meso >= 10000000 {
		fee = (meso * 4) / 100
	} else if meso >= 5000000 {
		fee = (meso * 3) / 100
	} else if meso >= 1000000 {
		fee = (meso * 18) / 1000
	} else if meso >= 100000 {
		fee = (meso * 8) / 1000
	}
	return fee
}
