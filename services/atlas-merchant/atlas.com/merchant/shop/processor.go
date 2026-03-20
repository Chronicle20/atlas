package shop

import (
	"atlas-merchant/frederick"
	asset2 "atlas-merchant/kafka/message/asset"
	character "atlas-merchant/kafka/message/character"
	"atlas-merchant/kafka/message/compartment"
	message "atlas-merchant/kafka/message"
	merchant "atlas-merchant/kafka/message/merchant"
	kafkaProducer "atlas-merchant/kafka/producer"
	"atlas-merchant/listing"
	msg "atlas-merchant/message"
	"atlas-merchant/visitor"
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-constants/item"
	"github.com/Chronicle20/atlas-constants/world"
	database "github.com/Chronicle20/atlas-database"
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
	GetByField(worldId world.Id, channelId channel.Id, mapId uint32, instanceId uuid.UUID) ([]Model, error)
	GetAllOpen() ([]Model, error)
	GetListingCounts(shopIds []uuid.UUID) (map[uuid.UUID]int64, error)
	SearchListingsByItemId(itemId uint32) ([]ListingSearchResult, error)
	GetListings(shopId uuid.UUID) ([]listing.Model, error)
	CreateShop(characterId uint32, shopType ShopType, title string, worldId world.Id, channelId channel.Id, mapId uint32, instanceId uuid.UUID, x int16, y int16, permitItemId uint32) (Model, error)
	OpenShop(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32) error
	EnterMaintenance(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32) error
	ExitMaintenance(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32) error
	CloseShop(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32, reason CloseReason) error
	GetExpired() ([]Model, error)
	AddListing(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32, itemId uint32, itemType byte, bundleSize uint16, bundleCount uint16, pricePerBundle uint32, itemSnapshot asset2.AssetData, inventoryType byte, assetId uint32) (listing.Model, error)
	RemoveListing(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32, listingIndex uint16) (listing.Model, error)
	UpdateListing(shopId uuid.UUID, listingIndex uint16, pricePerBundle uint32, bundleSize uint16, bundleCount uint16) error
	EnterShop(mb *message.Buffer) func(characterId uint32, shopId uuid.UUID) error
	ExitShop(mb *message.Buffer) func(characterId uint32, shopId uuid.UUID) error
	EjectAllVisitors(shopId uuid.UUID) ([]uint32, error)
	GetVisitors(shopId uuid.UUID) ([]uint32, error)
	GetShopForCharacter(characterId uint32) (uuid.UUID, error)
	PurchaseBundle(mb *message.Buffer) func(buyerCharacterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16, worldId world.Id) (PurchaseResult, error)
	SendMessage(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32, content string) error
	RetrieveFrederick(mb *message.Buffer) func(characterId uint32, worldId world.Id) error
	OpenShopAndEmit(shopId uuid.UUID, characterId uint32) error
	CloseShopAndEmit(shopId uuid.UUID, characterId uint32, reason CloseReason) error
	EnterMaintenanceAndEmit(shopId uuid.UUID, characterId uint32) error
	ExitMaintenanceAndEmit(shopId uuid.UUID, characterId uint32) error
	EnterShopAndEmit(characterId uint32, shopId uuid.UUID) error
	ExitShopAndEmit(characterId uint32, shopId uuid.UUID) error
	AddListingAndEmit(shopId uuid.UUID, characterId uint32, itemId uint32, itemType byte, bundleSize uint16, bundleCount uint16, pricePerBundle uint32, itemSnapshot asset2.AssetData, inventoryType byte, assetId uint32) (listing.Model, error)
	RemoveListingAndEmit(shopId uuid.UUID, characterId uint32, listingIndex uint16) (listing.Model, error)
	PurchaseBundleAndEmit(buyerCharacterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16, worldId world.Id) (PurchaseResult, error)
	SendMessageAndEmit(shopId uuid.UUID, characterId uint32, content string) error
	RetrieveFrederickAndEmit(characterId uint32, worldId world.Id) error
}

type PurchaseResult struct {
	ListingId        uuid.UUID
	ItemId           uint32
	ItemType         byte
	ItemSnapshot     asset2.AssetData
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
	Listing   listing.Model
	ShopId    uuid.UUID
	Title     string
	WorldId   world.Id
	ChannelId channel.Id
	MapId     uint32
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

func (p *ProcessorImpl) GetByField(worldId world.Id, channelId channel.Id, mapId uint32, instanceId uuid.UUID) ([]Model, error) {
	return model.SliceMap(Make)(getByField(worldId, channelId, mapId, instanceId)(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}

func (p *ProcessorImpl) GetAllOpen() ([]Model, error) {
	return model.SliceMap(Make)(getAllOpen()(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}

func (p *ProcessorImpl) GetListingCounts(shopIds []uuid.UUID) (map[uuid.UUID]int64, error) {
	return listing.NewProcessor(p.db.WithContext(p.ctx)).CountByShopIds(shopIds)
}

func (p *ProcessorImpl) SearchListingsByItemId(itemId uint32) ([]ListingSearchResult, error) {
	return searchListingsByItemId(itemId)(p.db.WithContext(p.ctx))()
}

func (p *ProcessorImpl) GetListings(shopId uuid.UUID) ([]listing.Model, error) {
	return listing.NewProcessor(p.db.WithContext(p.ctx)).GetByShopId(shopId)
}

func (p *ProcessorImpl) GetExpired() ([]Model, error) {
	return model.SliceMap(Make)(getExpired()(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}

func (p *ProcessorImpl) CreateShop(characterId uint32, shopType ShopType, title string, worldId world.Id, channelId channel.Id, mapId uint32, instanceId uuid.UUID, x int16, y int16, permitItemId uint32) (Model, error) {
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
		return p.GetByField(worldId, channelId, mapId, instanceId)
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
		WorldId:      worldId,
		ChannelId:    channelId,
		MapId:        mapId,
		InstanceId:   instanceId,
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
			ShopId:     id,
			ShopType:   shopType,
			WorldId:    worldId,
			ChannelId:  channelId,
			MapId:      mapId,
			InstanceId: instanceId,
		})
	}

	p.l.Infof("Shop [%s] created for character [%d], type [%d].", id, characterId, shopType)
	return m, nil
}

func (p *ProcessorImpl) OpenShop(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32) error {
	return func(shopId uuid.UUID, characterId uint32) error {
		var mapId uint32
		err := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			e, err := getById(shopId)(tx)()
			if err != nil {
				return err
			}

			if State(e.State) != Draft {
				return fmt.Errorf("%w: cannot open shop in state %d", ErrInvalidTransition, e.State)
			}

			lp := listing.NewProcessor(tx)
			count, err := lp.CountByShopId(shopId)
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

		p.addToMapIndex(mapId, shopId)

		m, err := p.GetById(shopId)
		if err != nil {
			return err
		}

		return mb.Put(merchant.EnvStatusEventTopic, StatusEventShopOpenedProvider(characterId, shopId, m))
	}
}

func (p *ProcessorImpl) EnterMaintenance(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32) error {
	return func(shopId uuid.UUID, characterId uint32) error {
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

		// Get visitor list with slots before ejection, then emit events.
		visitors, _ := p.GetVisitors(shopId)
		ejected, _ := p.EjectAllVisitors(shopId)
		emitEjectionEvents(mb, visitors, shopId)
		if len(ejected) > 0 {
			p.l.Infof("Shop [%s] entered maintenance, ejected %d visitors.", shopId, len(ejected))
		}

		return mb.Put(merchant.EnvStatusEventTopic, StatusEventMaintenanceEnteredProvider(characterId, shopId))
	}
}

func (p *ProcessorImpl) ExitMaintenance(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32) error {
	return func(shopId uuid.UUID, characterId uint32) error {
		var closed bool
		var mapId uint32
		err := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			e, err := getById(shopId)(tx)()
			if err != nil {
				return err
			}

			if State(e.State) != Maintenance {
				return fmt.Errorf("%w: cannot exit maintenance in state %d", ErrInvalidTransition, e.State)
			}

			lp := listing.NewProcessor(tx)
			count, err := lp.CountByShopId(shopId)
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
			mapId = e.MapId
			return nil
		})
		if err != nil {
			return err
		}

		if closed {
			p.removeFromRegistry(characterId)
			p.removeFromMapIndex(mapId, shopId)
			return mb.Put(merchant.EnvStatusEventTopic, StatusEventShopClosedProvider(characterId, shopId, CloseReasonEmpty))
		}
		return mb.Put(merchant.EnvStatusEventTopic, StatusEventMaintenanceExitedProvider(characterId, shopId))
	}
}

func (p *ProcessorImpl) CloseShop(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32, reason CloseReason) error {
	return func(shopId uuid.UUID, characterId uint32, reason CloseReason) error {
		m, err := p.GetById(shopId)
		if err != nil {
			return err
		}

		// For character shops (non-disconnect), snapshot listings for item return.
		var listings []listingSnapshot
		if m.ShopType() == CharacterShop && reason != CloseReasonDisconnect {
			ls, _ := p.GetListings(shopId)
			for _, l := range ls {
				listings = append(listings, listingSnapshot{
					ItemId:       l.ItemId(),
					ItemType:     l.ItemType(),
					Quantity:     l.Quantity(),
					ItemSnapshot: l.ItemSnapshot(),
				})
			}
		}

		var mapId uint32
		var shopType ShopType
		var mesoBalance uint32
		err = database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
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
		// Get visitor list with slots before ejection, then emit events.
		visitors, _ := p.GetVisitors(shopId)
		p.EjectAllVisitors(shopId)
		emitEjectionEvents(mb, visitors, shopId)

		if shopType == HiredMerchant {
			p.storeToFrederick(shopId, characterId, mesoBalance)
		}

		// Return items to character shop owner's inventory.
		for _, ls := range listings {
			acceptItemToBuffer(mb, characterId, ls)
		}

		p.l.Infof("Shop [%s] closed, reason [%d].", shopId, reason)
		return mb.Put(merchant.EnvStatusEventTopic, StatusEventShopClosedProvider(characterId, shopId, reason))
	}
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

func (p *ProcessorImpl) AddListing(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32, itemId uint32, itemType byte, bundleSize uint16, bundleCount uint16, pricePerBundle uint32, itemSnapshot asset2.AssetData, inventoryType byte, assetId uint32) (listing.Model, error) {
	return func(shopId uuid.UUID, characterId uint32, itemId uint32, itemType byte, bundleSize uint16, bundleCount uint16, pricePerBundle uint32, itemSnapshot asset2.AssetData, inventoryType byte, assetId uint32) (listing.Model, error) {
		if pricePerBundle == 0 {
			return listing.Model{}, errors.New("pricePerBundle must be at least 1")
		}
		if bundleSize == 0 {
			return listing.Model{}, errors.New("bundleSize must be at least 1")
		}
		if bundleCount == 0 {
			return listing.Model{}, errors.New("bundleCount must be at least 1")
		}

		if err := IsListableItem(itemId, itemSnapshot.Flag); err != nil {
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

			lp := listing.NewProcessor(tx)
			count, err := lp.CountByShopId(shopId)
			if err != nil {
				return err
			}
			if count >= MaxListings {
				return ErrListingLimitReached
			}

			result, err = lp.Create(shopId, p.t.Id(), itemId, itemType, bundleSize, bundleCount, pricePerBundle, itemSnapshot, uint16(count))
			return err
		})
		if err != nil {
			return listing.Model{}, err
		}

		quantity := uint32(bundleSize) * uint32(bundleCount)
		transactionId := uuid.New()
		if err := mb.Put(compartment.EnvCommandTopic, ReleaseAssetCommandProvider(transactionId, characterId, inventoryType, assetId, quantity)); err != nil {
			return result, err
		}

		return result, nil
	}
}

func (p *ProcessorImpl) RemoveListing(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32, listingIndex uint16) (listing.Model, error) {
	return func(shopId uuid.UUID, characterId uint32, listingIndex uint16) (listing.Model, error) {
		var result listing.Model
		err := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			e, err := getById(shopId)(tx)()
			if err != nil {
				return err
			}

			if State(e.State) != Draft && State(e.State) != Maintenance {
				return fmt.Errorf("%w: cannot remove listing in state %d", ErrInvalidTransition, e.State)
			}

			lp := listing.NewProcessor(tx)
			result, err = lp.GetByShopIdAndDisplayOrder(shopId, listingIndex)
			if err != nil {
				return err
			}

			if err = lp.Delete(result.Id()); err != nil {
				return err
			}

			return lp.DecrementDisplayOrderAfter(shopId, listingIndex)
		})
		if err != nil {
			return listing.Model{}, err
		}

		// Return item to owner's inventory.
		acceptItemToBuffer(mb, characterId, listingSnapshot{
			ItemId:       result.ItemId(),
			ItemType:     result.ItemType(),
			Quantity:     result.Quantity(),
			ItemSnapshot: result.ItemSnapshot(),
		})

		return result, nil
	}
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

		lp := listing.NewProcessor(tx)
		li, err := lp.GetByShopIdAndDisplayOrder(shopId, listingIndex)
		if err != nil {
			return err
		}

		return lp.UpdateFields(li.Id(), pricePerBundle, bundleSize, bundleCount)
	})
}

func (p *ProcessorImpl) PurchaseBundle(mb *message.Buffer) func(buyerCharacterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16, worldId world.Id) (PurchaseResult, error) {
	return func(buyerCharacterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16, worldId world.Id) (PurchaseResult, error) {
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

			lp := listing.NewProcessor(tx)
			li, err := lp.GetByShopIdAndDisplayOrder(shopId, listingIndex)
			if err != nil {
				return err
			}

			if li.BundlesRemaining() < bundleCount {
				return ErrInsufficientBundles
			}

			totalCost := int64(bundleCount) * int64(li.PricePerBundle())
			fee := GetFee(totalCost)

			result.ListingId = li.Id()
			result.ItemId = li.ItemId()
			result.ItemType = li.ItemType()
			result.ItemSnapshot = li.ItemSnapshot()
			result.BundleSize = li.BundleSize()
			result.BundlesPurchased = bundleCount
			result.TotalCost = totalCost
			result.Fee = fee
			result.NetAmount = totalCost - fee

			newBundlesRemaining := li.BundlesRemaining() - bundleCount
			newQuantity := li.BundleSize() * newBundlesRemaining

			rowsAffected, err := lp.UpdateBundles(li.Id(), newBundlesRemaining, newQuantity, li.Version())
			if err != nil {
				return err
			}
			if rowsAffected == 0 {
				return ErrVersionConflict
			}

			result.BundlesRemaining = newBundlesRemaining

			if newBundlesRemaining == 0 {
				if err = lp.Delete(li.Id()); err != nil {
					return err
				}

				if err = lp.DecrementDisplayOrderAfter(shopId, listingIndex); err != nil {
					return err
				}

				count, err := lp.CountByShopId(shopId)
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

			if ShopType(e.ShopType) == HiredMerchant {
				e.MesoBalance += uint32(totalCost - fee)
			}

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
			// Get visitor list with slots before ejection, then emit events.
			soldOutVisitors, _ := p.GetVisitors(shopId)
			p.EjectAllVisitors(shopId)
			emitEjectionEvents(mb, soldOutVisitors, shopId)
			p.l.Infof("Shop [%s] sold out and closed.", shopId)
		}

		// Deduct mesos from buyer.
		transactionId := uuid.New()
		if err := mb.Put(character.EnvCommandTopic, ChangeMesoCommandProvider(transactionId, worldId, buyerCharacterId, result.ShopOwnerId, "MERCHANT", -int32(result.TotalCost))); err != nil {
			return result, err
		}

		// Grant items to buyer.
		ad := result.ItemSnapshot.WithQuantity(uint32(result.BundleSize) * uint32(result.BundlesPurchased))
		invType, ok := inventory.TypeFromItemId(item.Id(result.ItemId))
		if ok {
			itemTransactionId := uuid.New()
			_ = mb.Put(compartment.EnvCommandTopic, AcceptAssetCommandProvider(itemTransactionId, buyerCharacterId, byte(invType), result.ItemId, ad))
		}

		// Credit mesos to owner (character shops only).
		if result.ShopType == CharacterShop && result.NetAmount > 0 {
			creditTransactionId := uuid.New()
			_ = mb.Put(character.EnvCommandTopic, ChangeMesoCommandProvider(creditTransactionId, worldId, result.ShopOwnerId, buyerCharacterId, "MERCHANT", int32(result.NetAmount)))
		}

		_ = mb.Put(merchant.EnvListingEventTopic, ListingEventPurchasedProvider(shopId, listingIndex, buyerCharacterId, bundleCount, result.BundlesRemaining))

		if result.ShopClosed {
			_ = mb.Put(merchant.EnvStatusEventTopic, StatusEventShopClosedProvider(result.ShopOwnerId, shopId, CloseReasonSoldOut))
		}

		return result, nil
	}
}

func (p *ProcessorImpl) EnterShop(mb *message.Buffer) func(characterId uint32, shopId uuid.UUID) error {
	return func(characterId uint32, shopId uuid.UUID) error {
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
			return mb.Put(merchant.EnvStatusEventTopic, StatusEventCapacityFullProvider(characterId, shopId))
		}

		if err = vr.AddVisitor(p.ctx, p.t, shopId, characterId); err != nil {
			return err
		}

		// Compute slot based on insertion-ordered visitor list.
		visitors, err := vr.GetVisitors(p.ctx, p.t, shopId)
		if err != nil {
			return err
		}
		slot := visitorSlot(visitors, characterId)

		return mb.Put(merchant.EnvStatusEventTopic, StatusEventVisitorEnteredProvider(characterId, shopId, slot))
	}
}

func (p *ProcessorImpl) ExitShop(mb *message.Buffer) func(characterId uint32, shopId uuid.UUID) error {
	return func(characterId uint32, shopId uuid.UUID) error {
		vr := visitor.GetRegistry()
		if vr == nil {
			return errors.New("visitor registry not initialized")
		}

		// Compute slot before removal (sorted set guarantees insertion order).
		visitors, err := vr.GetVisitors(p.ctx, p.t, shopId)
		if err != nil {
			return err
		}
		slot := visitorSlot(visitors, characterId)

		if err := vr.RemoveVisitor(p.ctx, p.t, shopId, characterId); err != nil {
			return err
		}

		return mb.Put(merchant.EnvStatusEventTopic, StatusEventVisitorExitedProvider(characterId, shopId, slot))
	}
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

func (p *ProcessorImpl) GetShopForCharacter(characterId uint32) (uuid.UUID, error) {
	vr := visitor.GetRegistry()
	if vr == nil {
		return uuid.Nil, errors.New("visitor registry not initialized")
	}
	return vr.GetShopForCharacter(p.ctx, p.t, characterId)
}

// visitorSlot returns the 1-indexed slot for a visitor in the ordered visitor list.
// Slot 0 is reserved for the shop owner. Returns 0 if not found.
func visitorSlot(visitors []uint32, characterId uint32) byte {
	for i, v := range visitors {
		if v == characterId {
			return byte(i + 1)
		}
	}
	return 0
}

// emitEjectionEvents emits a VISITOR_EJECTED event for each visitor in the ordered list.
func emitEjectionEvents(mb *message.Buffer, visitors []uint32, shopId uuid.UUID) {
	for i, cid := range visitors {
		slot := byte(i + 1)
		_ = mb.Put(merchant.EnvStatusEventTopic, StatusEventVisitorEjectedProvider(cid, shopId, slot))
	}
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

func (p *ProcessorImpl) SendMessage(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32, content string) error {
	return func(shopId uuid.UUID, characterId uint32, content string) error {
		mp := msg.NewProcessor(p.l, p.ctx, p.db)
		if err := mp.SendMessage(shopId, characterId, content); err != nil {
			return err
		}

		visitors, err := p.GetVisitors(shopId)
		if err != nil {
			return err
		}

		s, err := p.GetById(shopId)
		if err != nil {
			return err
		}

		// Slot 0 is the owner. Visitors start at slot 1.
		var slot byte
		if characterId == s.CharacterId() {
			slot = 0
		} else {
			for i, v := range visitors {
				if v == characterId {
					slot = byte(i + 1)
					break
				}
			}
		}

		return mb.Put(merchant.EnvStatusEventTopic, StatusEventMessageSentProvider(characterId, shopId, slot, content))
	}
}

func (p *ProcessorImpl) RetrieveFrederick(mb *message.Buffer) func(characterId uint32, worldId world.Id) error {
	return func(characterId uint32, worldId world.Id) error {
		fp := frederick.NewProcessor(p.l, p.ctx, p.db)

		items, err := fp.GetItems(characterId)
		if err != nil {
			return err
		}

		mesos, err := fp.GetMesos(characterId)
		if err != nil {
			return err
		}

		if len(items) == 0 && len(mesos) == 0 {
			p.l.Debugf("No items or mesos at Frederick for character [%d].", characterId)
			return nil
		}

		// Transfer items to character's inventory.
		for _, fi := range items {
			ad := fi.ItemSnapshot().WithQuantity(uint32(fi.Quantity()))

			invType, ok := inventory.TypeFromItemId(item.Id(fi.ItemId()))
			if !ok {
				p.l.Errorf("Unable to determine inventory type for Frederick item [%d].", fi.ItemId())
				continue
			}

			transactionId := uuid.New()
			_ = mb.Put(compartment.EnvCommandTopic, AcceptAssetCommandProvider(transactionId, characterId, byte(invType), fi.ItemId(), ad))
		}

		// Transfer mesos to character.
		var totalMesos uint32
		for _, fm := range mesos {
			totalMesos += fm.Amount()
		}
		if totalMesos > 0 {
			transactionId := uuid.New()
			_ = mb.Put(character.EnvCommandTopic, ChangeMesoCommandProvider(transactionId, worldId, characterId, 0, "FREDERICK", int32(totalMesos)))
		}

		// Clear Frederick storage and notifications.
		if err := fp.ClearItems(characterId); err != nil {
			p.l.WithError(err).Errorf("Error clearing Frederick items for character [%d].", characterId)
		}
		if err := fp.ClearMesos(characterId); err != nil {
			p.l.WithError(err).Errorf("Error clearing Frederick mesos for character [%d].", characterId)
		}
		if err := fp.ClearNotifications(characterId); err != nil {
			p.l.WithError(err).Errorf("Error clearing Frederick notifications for character [%d].", characterId)
		}

		p.l.Infof("Retrieved %d items and %d meso records from Frederick for character [%d].", len(items), len(mesos), characterId)
		return nil
	}
}

// AndEmit wrappers.

func (p *ProcessorImpl) OpenShopAndEmit(shopId uuid.UUID, characterId uint32) error {
	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		return p.OpenShop(buf)(shopId, characterId)
	})
}

func (p *ProcessorImpl) CloseShopAndEmit(shopId uuid.UUID, characterId uint32, reason CloseReason) error {
	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		return p.CloseShop(buf)(shopId, characterId, reason)
	})
}

func (p *ProcessorImpl) EnterMaintenanceAndEmit(shopId uuid.UUID, characterId uint32) error {
	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		return p.EnterMaintenance(buf)(shopId, characterId)
	})
}

func (p *ProcessorImpl) ExitMaintenanceAndEmit(shopId uuid.UUID, characterId uint32) error {
	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		return p.ExitMaintenance(buf)(shopId, characterId)
	})
}

func (p *ProcessorImpl) EnterShopAndEmit(characterId uint32, shopId uuid.UUID) error {
	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		return p.EnterShop(buf)(characterId, shopId)
	})
}

func (p *ProcessorImpl) ExitShopAndEmit(characterId uint32, shopId uuid.UUID) error {
	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		return p.ExitShop(buf)(characterId, shopId)
	})
}

func (p *ProcessorImpl) AddListingAndEmit(shopId uuid.UUID, characterId uint32, itemId uint32, itemType byte, bundleSize uint16, bundleCount uint16, pricePerBundle uint32, itemSnapshot asset2.AssetData, inventoryType byte, assetId uint32) (listing.Model, error) {
	var result listing.Model
	err := message.Emit(p.producer)(func(buf *message.Buffer) error {
		var err error
		result, err = p.AddListing(buf)(shopId, characterId, itemId, itemType, bundleSize, bundleCount, pricePerBundle, itemSnapshot, inventoryType, assetId)
		return err
	})
	return result, err
}

func (p *ProcessorImpl) RemoveListingAndEmit(shopId uuid.UUID, characterId uint32, listingIndex uint16) (listing.Model, error) {
	var result listing.Model
	err := message.Emit(p.producer)(func(buf *message.Buffer) error {
		var err error
		result, err = p.RemoveListing(buf)(shopId, characterId, listingIndex)
		return err
	})
	return result, err
}

func (p *ProcessorImpl) PurchaseBundleAndEmit(buyerCharacterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16, worldId world.Id) (PurchaseResult, error) {
	var result PurchaseResult
	err := message.Emit(p.producer)(func(buf *message.Buffer) error {
		var err error
		result, err = p.PurchaseBundle(buf)(buyerCharacterId, shopId, listingIndex, bundleCount, worldId)
		return err
	})
	return result, err
}

func (p *ProcessorImpl) SendMessageAndEmit(shopId uuid.UUID, characterId uint32, content string) error {
	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		return p.SendMessage(buf)(shopId, characterId, content)
	})
}

func (p *ProcessorImpl) RetrieveFrederickAndEmit(characterId uint32, worldId world.Id) error {
	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		return p.RetrieveFrederick(buf)(characterId, worldId)
	})
}

// listingSnapshot captures listing data before shop closure.
type listingSnapshot struct {
	ItemId       uint32
	ItemType     byte
	Quantity     uint16
	ItemSnapshot asset2.AssetData
}

func acceptItemToBuffer(buf *message.Buffer, characterId uint32, ls listingSnapshot) {
	ad := ls.ItemSnapshot.WithQuantity(uint32(ls.Quantity))

	invType, ok := inventory.TypeFromItemId(item.Id(ls.ItemId))
	if !ok {
		return
	}

	transactionId := uuid.New()
	_ = buf.Put(compartment.EnvCommandTopic, AcceptAssetCommandProvider(transactionId, characterId, byte(invType), ls.ItemId, ad))
}

// IsShopFull checks if the error is a shop capacity error.
func IsShopFull(err error) bool {
	return errors.Is(err, ErrShopFull)
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
