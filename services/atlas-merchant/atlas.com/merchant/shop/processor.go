package shop

import (
	"atlas-merchant/frederick"
	message "atlas-merchant/kafka/message"
	asset2 "atlas-merchant/kafka/message/asset"
	character "atlas-merchant/kafka/message/character"
	"atlas-merchant/kafka/message/compartment"
	merchant "atlas-merchant/kafka/message/merchant"
	kafkaProducer "atlas-merchant/kafka/producer"
	"atlas-merchant/blacklist"
	"atlas-merchant/listing"
	msg "atlas-merchant/message"
	"atlas-merchant/visit"
	"atlas-merchant/visitor"
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const MaxListings = 16
const MaxVisitors = 3

// MaxSearchResults caps the shop-scanner search (client renders at most
// 200 rows — SP_3630/3631, task-127 design §1.4).
const MaxSearchResults = 200

type Processor interface {
	WithTransaction(tx *gorm.DB) Processor
	GetById(id uuid.UUID) (Model, error)
	ByIdProvider(id uuid.UUID) model.Provider[Model]
	GetByCharacterId(characterId uint32) ([]Model, error)
	GetByField(worldId world.Id, channelId channel.Id, mapId uint32, instanceId uuid.UUID) ([]Model, error)
	GetAllOpen() ([]Model, error)
	GetListingCounts(shopIds []uuid.UUID) (map[uuid.UUID]int64, error)
	SearchListingsByItemId(criteria ListingSearchCriteria) ([]ListingSearchResult, error)
	GetListings(shopId uuid.UUID) ([]listing.Model, error)
	CreateShop(characterId uint32, shopType ShopType, title string, worldId world.Id, channelId channel.Id, mapId uint32, instanceId uuid.UUID, x int16, y int16, permitItemId uint32) (Model, error)
	CreateShopAndEmit(characterId uint32, shopType ShopType, title string, worldId world.Id, channelId channel.Id, mapId uint32, instanceId uuid.UUID, x int16, y int16, permitItemId uint32) (Model, error)
	OpenShop(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32) error
	EnterMaintenance(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32) error
	ExitMaintenance(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32) error
	CloseShop(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32, reason CloseReason) error
	GetExpired() ([]Model, error)
	AddListing(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32, itemId uint32, itemType byte, bundleSize uint16, bundleCount uint16, pricePerBundle uint32, itemSnapshot asset2.AssetData, inventoryType byte, assetId uint32) (listing.Model, error)
	RemoveListing(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32, listingIndex uint16) (listing.Model, error)
	UpdateListing(shopId uuid.UUID, listingIndex uint16, pricePerBundle uint32, bundleSize uint16, bundleCount uint16) error
	WithdrawMeso(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32) error
	OrganizeListings(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32) error
	WithdrawMesoAndEmit(shopId uuid.UUID, characterId uint32) error
	OrganizeListingsAndEmit(shopId uuid.UUID, characterId uint32) error
	EnterShop(mb *message.Buffer) func(characterId uint32, shopId uuid.UUID, visitorName string) error
	AddToBlacklist(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32, name string, bannedCharacterId uint32) error
	RemoveFromBlacklist(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32, name string) error
	GetBlacklist(shopId uuid.UUID) ([]string, error)
	GetVisits(shopId uuid.UUID) ([]visit.Model, error)
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
	EnterShopAndEmit(characterId uint32, shopId uuid.UUID, visitorName string) error
	AddToBlacklistAndEmit(shopId uuid.UUID, characterId uint32, name string, bannedCharacterId uint32) error
	RemoveFromBlacklistAndEmit(shopId uuid.UUID, characterId uint32, name string) error
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

// ListingSearchCriteria narrows a listing search. WorldId nil means
// tenant-wide (pre-task-127 behavior); the owl path always sets it.
type ListingSearchCriteria struct {
	ItemId     uint32
	WorldId    *world.Id
	Descending bool
}

type ListingSearchResult struct {
	Listing     listing.Model
	ShopId      uuid.UUID
	Title       string
	WorldId     world.Id
	ChannelId   channel.Id
	MapId       uint32
	ShopOwnerId uint32
	ShopType    ShopType
	State       State
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
var ErrNotOwner = errors.New("character does not own shop")

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

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) WithTransaction(tx *gorm.DB) Processor {
	return &ProcessorImpl{
		l:        p.l,
		ctx:      p.ctx,
		db:       tx,
		t:        p.t,
		producer: p.producer,
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

func (p *ProcessorImpl) SearchListingsByItemId(criteria ListingSearchCriteria) ([]ListingSearchResult, error) {
	return searchListingsByItemId(p.t.Id(), criteria)(p.db.WithContext(p.ctx))()
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

// CreateShopAndEmit places a shop and, on a player-facing validation failure,
// emits a SHOP_CREATE_FAILED status event so the channel can tell the player
// why nothing opened (a mini-room error notice). Placement failures
// short-circuit before any DB write, so the feedback is emitted in its own
// committed transaction rather than being rolled back with the (empty) failure.
func (p *ProcessorImpl) CreateShopAndEmit(characterId uint32, shopType ShopType, title string, worldId world.Id, channelId channel.Id, mapId uint32, instanceId uuid.UUID, x int16, y int16, permitItemId uint32) (Model, error) {
	m, err := p.CreateShop(characterId, shopType, title, worldId, channelId, mapId, instanceId, x, y, permitItemId)
	if err == nil {
		// Success: the shop is Draft. Drop the owner into the shop UI so they
		// can stock it before the formal open. Emitted in its own tx after the
		// create commit.
		if emitErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
				return buf.Put(merchant.EnvStatusEventTopic, StatusEventShopSetupProvider(characterId, m.Id(), m))
			})
		}); emitErr != nil {
			p.l.WithError(emitErr).Warnf("Unable to emit shop-setup for character [%d].", characterId)
		}
		return m, nil
	}

	reason := shopCreateFailureReason(err)
	if reason == "" {
		return m, err
	}

	if emitErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return buf.Put(merchant.EnvStatusEventTopic, StatusEventShopCreateFailedProvider(characterId, worldId, channelId, reason))
		})
	}); emitErr != nil {
		p.l.WithError(emitErr).Warnf("Unable to emit shop-create-failed feedback for character [%d].", characterId)
	}
	return m, err
}

// shopCreateFailureReason maps a CreateShop error to a player-facing feedback
// reason, or "" for internal errors that warrant no client message.
func shopCreateFailureReason(err error) string {
	switch {
	case errors.Is(err, ErrTooCloseToPortal):
		return merchant.ShopCreateFailReasonTooCloseToPortal
	case errors.Is(err, ErrTooCloseToShop):
		return merchant.ShopCreateFailReasonTooCloseToShop
	case errors.Is(err, ErrNotFreemarketRoom):
		return merchant.ShopCreateFailReasonNotFreeMarket
	case errors.Is(err, ErrShopLimitReached), errors.Is(err, ErrFrederickPending):
		return merchant.ShopCreateFailReasonUnable
	default:
		return ""
	}
}

// requireOwner rejects an owner-only mutation issued by anyone but the
// shop's owner. Commands arrive over Kafka, which trusts the producer's
// characterId — this is the server-side backstop behind the channel's gating.
func requireOwner(e Entity, characterId uint32) error {
	if e.CharacterId != characterId {
		return fmt.Errorf("%w: shop [%s] belongs to [%d], actor [%d]", ErrNotOwner, e.Id, e.CharacterId, characterId)
	}
	return nil
}

func (p *ProcessorImpl) OpenShop(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32) error {
	return func(shopId uuid.UUID, characterId uint32) error {
		var mapId uint32
		err := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			e, err := getById(shopId)(tx)()
			if err != nil {
				return err
			}

			if err = requireOwner(e, characterId); err != nil {
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

			if err = requireOwner(e, characterId); err != nil {
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
		// Owner opened the management/maintenance view: kick visitors with the
		// "shop is closed" message (no dedicated maintenance message exists).
		emitEjectionEvents(mb, visitors, shopId, merchant.LeaveReasonShopClosed)
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

			if err = requireOwner(e, characterId); err != nil {
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

		// Character shops return unsold items directly to the owner's inventory on
		// EVERY close, including disconnect/logout. The AcceptAsset command updates
		// the compartment store regardless of the owner's session, so an offline
		// owner still gets them back; excluding disconnect orphaned the items on
		// the closed shop with no Fredrick fallback (task-127). Hired merchants
		// deposit with Fredrick instead (storeToFrederick, below).
		var listings []listingSnapshot
		if m.ShopType() == CharacterShop {
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

			if err = requireOwner(e, characterId); err != nil {
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
		// A sold-out close reports out-of-stock; every other close (manual,
		// disconnect, expired, empty) reports the shop is closed.
		leaveReason := merchant.LeaveReasonShopClosed
		if reason == CloseReasonSoldOut {
			leaveReason = merchant.LeaveReasonOutOfStock
		}
		emitEjectionEvents(mb, visitors, shopId, leaveReason)

		if shopType == HiredMerchant {
			if err := p.storeToFrederick(shopId, characterId, mesoBalance); err != nil {
				return err
			}
		}

		// Return items to character shop owner's inventory.
		for _, ls := range listings {
			acceptItemToBuffer(mb, characterId, ls)
		}

		p.l.Infof("Shop [%s] closed, reason [%d].", shopId, reason)
		return mb.Put(merchant.EnvStatusEventTopic, StatusEventShopClosedProvider(characterId, shopId, reason))
	}
}

// storeToFrederick persists unsold listing items and meso balance to
// Frederick storage on shop close. Returns an error on the first failed
// write so the caller (CloseShop, inside CloseShopAndEmit's outer tx) can
// abort the closure — a partial/failed store must NOT let the shop-closed
// event enqueue to the outbox, otherwise unsold items/mesos would silently
// vanish while the client is told the shop closed cleanly.
func (p *ProcessorImpl) storeToFrederick(shopId uuid.UUID, characterId uint32, mesoBalance uint32) error {
	listings, err := p.GetListings(shopId)
	if err != nil {
		p.l.WithError(err).Errorf("Error retrieving listings for Frederick storage, shop [%s].", shopId)
		return err
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
			return err
		}
	}

	if mesoBalance > 0 {
		if err := fp.StoreMesos(characterId, mesoBalance); err != nil {
			p.l.WithError(err).Errorf("Error storing mesos to Frederick for character [%d].", characterId)
			return err
		}
	}

	// Create notification for Frederick retrieval reminders.
	if len(listings) > 0 || mesoBalance > 0 {
		if err := fp.CreateNotification(characterId); err != nil {
			p.l.WithError(err).Errorf("Error creating Frederick notification for character [%d].", characterId)
			return err
		}
	}

	return nil
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

			if err = requireOwner(e, characterId); err != nil {
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

		// Refresh the owner's store view (UPDATE_MERCHANT); without this the
		// client that dropped an item into a slot never gets a reply and freezes.
		if err := mb.Put(merchant.EnvStatusEventTopic, StatusEventShopUpdatedProvider(characterId, shopId)); err != nil {
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

			if err = requireOwner(e, characterId); err != nil {
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

		// Refresh the owner's store view after pulling the item back.
		if err := mb.Put(merchant.EnvStatusEventTopic, StatusEventShopUpdatedProvider(characterId, shopId)); err != nil {
			return result, err
		}

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

// WithdrawMeso credits the hired merchant's accrued sale balance to the owner
// and zeroes it (MERCHANT_WITHDRAW_MESO). Owner-only; only hired merchants
// accrue a balance (personal shops pay the owner per sale). Emits SHOP_UPDATED
// so the owner's management view refreshes to meso 0.
func (p *ProcessorImpl) WithdrawMeso(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32) error {
	return func(shopId uuid.UUID, characterId uint32) error {
		var amount uint32
		var worldId world.Id
		err := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			e, err := getById(shopId)(tx)()
			if err != nil {
				return err
			}
			if err = requireOwner(e, characterId); err != nil {
				return err
			}
			if ShopType(e.ShopType) != HiredMerchant {
				return fmt.Errorf("%w: only hired merchants accrue meso", ErrInvalidTransition)
			}
			amount = e.MesoBalance
			worldId = e.WorldId
			if amount == 0 {
				return nil
			}
			e.MesoBalance = 0
			_, err = update(&e)(tx)()
			return err
		})
		if err != nil {
			return err
		}

		if amount > 0 {
			transactionId := uuid.New()
			if err := mb.Put(character.EnvCommandTopic, ChangeMesoCommandProvider(transactionId, worldId, characterId, 0, "MERCHANT", int32(amount))); err != nil {
				return err
			}
		}
		return mb.Put(merchant.EnvStatusEventTopic, StatusEventShopUpdatedProvider(characterId, shopId))
	}
}

// OrganizeListings compacts the shop's listing display: sold-out rows
// (0 bundles remaining) are dropped and the survivors are renumbered from 0
// (MERCHANT_ORGANIZE). Owner-only. An empty result closes the shop
// (CloseReasonEmpty); otherwise SHOP_UPDATED refreshes the view.
func (p *ProcessorImpl) OrganizeListings(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32) error {
	return func(shopId uuid.UUID, characterId uint32) error {
		var remaining int
		err := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
			e, err := getById(shopId)(tx)()
			if err != nil {
				return err
			}
			if err = requireOwner(e, characterId); err != nil {
				return err
			}

			lp := listing.NewProcessor(tx)
			listings, err := lp.GetByShopId(shopId)
			if err != nil {
				return err
			}

			order := uint16(0)
			for _, li := range listings {
				if li.BundlesRemaining() == 0 {
					if err := lp.Delete(li.Id()); err != nil {
						return err
					}
					continue
				}
				if li.DisplayOrder() != order {
					if err := lp.SetDisplayOrder(li.Id(), order); err != nil {
						return err
					}
				}
				order++
			}
			remaining = int(order)
			return nil
		})
		if err != nil {
			return err
		}

		if remaining == 0 {
			return p.CloseShop(mb)(shopId, characterId, CloseReasonEmpty)
		}
		return mb.Put(merchant.EnvStatusEventTopic, StatusEventShopUpdatedProvider(characterId, shopId))
	}
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
			emitEjectionEvents(mb, soldOutVisitors, shopId, merchant.LeaveReasonOutOfStock)
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

func (p *ProcessorImpl) EnterShop(mb *message.Buffer) func(characterId uint32, shopId uuid.UUID, visitorName string) error {
	return func(characterId uint32, shopId uuid.UUID, visitorName string) error {
		e, err := getById(shopId)(p.db.WithContext(p.ctx))()
		if err != nil {
			return err
		}

		if State(e.State) != Open {
			// Surface the faithful client error rather than a silent drop:
			// a shop being managed reads as under maintenance, anything else
			// (Draft/Closed) as room-closed.
			reason := merchant.EnterFailReasonRoomClosed
			if State(e.State) == Maintenance {
				reason = merchant.EnterFailReasonUndergoingMaintenance
			}
			return mb.Put(merchant.EnvStatusEventTopic, StatusEventEnterFailedProvider(characterId, shopId, reason))
		}

		// Blacklist enforcement (by name, Cosmic-faithful): a banned visitor
		// gets the "cannot enter" error rather than being admitted.
		if visitorName != "" {
			banned, err := blacklist.NewProcessor(p.l, p.ctx, p.db).IsBlacklisted(shopId, visitorName)
			if err != nil {
				p.l.WithError(err).Warnf("Unable to check blacklist for shop [%s] visitor [%s].", shopId, visitorName)
			} else if banned {
				return mb.Put(merchant.EnvStatusEventTopic, StatusEventEnterFailedProvider(characterId, shopId, merchant.EnterFailReasonBlacklisted))
			}
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

		// Record the visit for the owner's visit-list (best-effort).
		if err := visit.NewProcessor(p.l, p.ctx, p.db).Record(shopId, visitorName); err != nil {
			p.l.WithError(err).Warnf("Unable to record visit for shop [%s] visitor [%s].", shopId, visitorName)
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

// GetShopForCharacter resolves the shop the character is currently occupying.
// Visitor registry first (a character inside someone else's shop), then owner
// occupancy: a character occupies their OWN shop while it is owner-attached —
// a personal shop in any non-Closed state, or a hired merchant in Draft
// (setup) or Maintenance (management). An Open hired merchant runs
// owner-detached and does not count. Without owner occupancy, every
// owner-side op the channel routes through /characters/{id}/visiting (OPEN,
// PUT_ITEM, EXIT, CHAT) 404s on a freshly created Draft shop.
func (p *ProcessorImpl) GetShopForCharacter(characterId uint32) (uuid.UUID, error) {
	vr := visitor.GetRegistry()
	if vr == nil {
		return uuid.Nil, errors.New("visitor registry not initialized")
	}
	if shopId, err := vr.GetShopForCharacter(p.ctx, p.t, characterId); err == nil {
		return shopId, nil
	} else if !errors.Is(err, atlasredis.ErrNotFound) {
		// Only a genuine miss falls through to owner occupancy — a transient
		// Redis failure must surface, not masquerade as "not visiting".
		return uuid.Nil, err
	}

	// Owner occupancy. The Redis activeShops entry is a fast-path cache, but the
	// DB is authoritative: a missing entry (eviction, an uncommitted CreateShop
	// Put, a close-desync) or a stale entry (pointing at a shop the owner no
	// longer occupies) must NOT strand the owner — every owner-side op routes
	// through here, and a false miss 404s /characters/{id}/visiting, freezing
	// the client (task-127). Trust the cache only when it still resolves to an
	// owner-occupied shop; otherwise fall back to the DB and re-seed.
	if r := GetRegistry(); r != nil {
		entry, err := r.activeShops.Get(p.ctx, p.t, characterId)
		if err == nil {
			if m, gerr := p.GetById(entry.ShopId); gerr == nil && isOwnerOccupied(m) {
				return entry.ShopId, nil
			}
			// Stale/closed cache entry — fall through to the DB.
		} else if !errors.Is(err, atlasredis.ErrNotFound) {
			return uuid.Nil, err
		}
	}

	m, err := model.Map(Make)(getOwnerOccupiedShop(characterId)(p.db.WithContext(p.ctx)))()
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return uuid.Nil, fmt.Errorf("character [%d] is not occupying a shop", characterId)
		}
		return uuid.Nil, err
	}
	p.seedOccupancy(characterId, m)
	return m.Id(), nil
}

// isOwnerOccupied reports whether an owner is currently occupying (and may act
// on) their own shop: a personal shop in any non-Closed state, or a hired
// merchant only in Draft/Maintenance (an Open hired merchant is owner-detached).
func isOwnerOccupied(m Model) bool {
	switch m.ShopType() {
	case CharacterShop:
		return m.State() != Closed
	case HiredMerchant:
		return m.State() == Draft || m.State() == Maintenance
	default:
		return false
	}
}

// seedOccupancy repopulates the activeShops fast-path cache from an
// authoritative model, best-effort (a cache write failure is non-fatal — the DB
// fallback covers the next lookup).
func (p *ProcessorImpl) seedOccupancy(characterId uint32, m Model) {
	r := GetRegistry()
	if r == nil {
		return
	}
	_ = r.activeShops.Put(p.ctx, p.t, characterId, ActiveShopEntry{
		ShopId:     m.Id(),
		ShopType:   m.ShopType(),
		WorldId:    m.WorldId(),
		ChannelId:  m.ChannelId(),
		MapId:      m.MapId(),
		InstanceId: m.InstanceId(),
	})
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

// emitEjectionEvents emits a VISITOR_EJECTED event for each visitor in the
// ordered list. leaveReason is the client leaveReason table key sent to each
// ejected visitor so their room UI shows the correct message (SHOP_CLOSED,
// OUT_OF_STOCK, USER_BANNED) rather than an empty dialog.
func emitEjectionEvents(mb *message.Buffer, visitors []uint32, shopId uuid.UUID, leaveReason string) {
	for i, cid := range visitors {
		slot := byte(i + 1)
		_ = mb.Put(merchant.EnvStatusEventTopic, StatusEventVisitorEjectedProvider(cid, shopId, slot, leaveReason))
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

		// Clear Frederick storage and notifications. A failure here must abort
		// the enclosing tx (RetrieveFrederickAndEmit) so the asset/meso grant
		// commands buffered above are NOT enqueued to the outbox — otherwise
		// the character would receive a duplicate grant on retry while the
		// items/mesos remain stuck at Frederick.
		if err := fp.ClearItems(characterId); err != nil {
			p.l.WithError(err).Errorf("Error clearing Frederick items for character [%d].", characterId)
			return err
		}
		if err := fp.ClearMesos(characterId); err != nil {
			p.l.WithError(err).Errorf("Error clearing Frederick mesos for character [%d].", characterId)
			return err
		}
		if err := fp.ClearNotifications(characterId); err != nil {
			p.l.WithError(err).Errorf("Error clearing Frederick notifications for character [%d].", characterId)
			return err
		}

		p.l.Infof("Retrieved %d items and %d meso records from Frederick for character [%d].", len(items), len(mesos), characterId)
		return nil
	}
}

// AndEmit wrappers.

func (p *ProcessorImpl) OpenShopAndEmit(shopId uuid.UUID, characterId uint32) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).OpenShop(buf)(shopId, characterId)
		})
	})
}

func (p *ProcessorImpl) WithdrawMesoAndEmit(shopId uuid.UUID, characterId uint32) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).WithdrawMeso(buf)(shopId, characterId)
		})
	})
}

func (p *ProcessorImpl) OrganizeListingsAndEmit(shopId uuid.UUID, characterId uint32) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).OrganizeListings(buf)(shopId, characterId)
		})
	})
}

func (p *ProcessorImpl) CloseShopAndEmit(shopId uuid.UUID, characterId uint32, reason CloseReason) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).CloseShop(buf)(shopId, characterId, reason)
		})
	})
}

func (p *ProcessorImpl) EnterMaintenanceAndEmit(shopId uuid.UUID, characterId uint32) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).EnterMaintenance(buf)(shopId, characterId)
		})
	})
}

func (p *ProcessorImpl) ExitMaintenanceAndEmit(shopId uuid.UUID, characterId uint32) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).ExitMaintenance(buf)(shopId, characterId)
		})
	})
}

// EnterShopAndEmit stays on the direct producer path: EnterShop only mutates
// the Redis visitor registry (no Postgres write), so there is no DB state
// change to couple the emit to.
func (p *ProcessorImpl) EnterShopAndEmit(characterId uint32, shopId uuid.UUID, visitorName string) error {
	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		return p.EnterShop(buf)(characterId, shopId, visitorName)
	})
}

// AddToBlacklist / RemoveFromBlacklist are owner-only. A banned name cannot
// enter the shop (EnterShop enforcement). Both emit SHOP_UPDATED-style refresh
// via the blacklist-view request path; here they only mutate + confirm.
func (p *ProcessorImpl) AddToBlacklist(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32, name string, bannedCharacterId uint32) error {
	return func(shopId uuid.UUID, characterId uint32, name string, bannedCharacterId uint32) error {
		e, err := getById(shopId)(p.db.WithContext(p.ctx))()
		if err != nil {
			return err
		}
		if err := requireOwner(e, characterId); err != nil {
			return err
		}
		if err := blacklist.NewProcessor(p.l, p.ctx, p.db).Add(shopId, name); err != nil {
			return err
		}
		// If the banned player is currently in the shop, eject them with the
		// USER_BANNED leave reason (the ban button kicks them out, not just a
		// name-only blacklist add). Re-entry is already blocked by the EnterShop
		// blacklist check.
		if bannedCharacterId != 0 {
			if vr := visitor.GetRegistry(); vr != nil {
				visitors, verr := vr.GetVisitors(p.ctx, p.t, shopId)
				if verr == nil {
					slot := visitorSlot(visitors, bannedCharacterId)
					if slot != 0 {
						if rerr := vr.RemoveVisitor(p.ctx, p.t, shopId, bannedCharacterId); rerr != nil {
							p.l.WithError(rerr).Warnf("Unable to eject banned visitor [%d] from shop [%s].", bannedCharacterId, shopId)
						} else if perr := mb.Put(merchant.EnvStatusEventTopic, StatusEventVisitorEjectedProvider(bannedCharacterId, shopId, slot, merchant.LeaveReasonUserBanned)); perr != nil {
							return perr
						}
					}
				}
			}
		}
		return mb.Put(merchant.EnvStatusEventTopic, StatusEventBlacklistUpdatedProvider(characterId, shopId))
	}
}

func (p *ProcessorImpl) RemoveFromBlacklist(mb *message.Buffer) func(shopId uuid.UUID, characterId uint32, name string) error {
	return func(shopId uuid.UUID, characterId uint32, name string) error {
		e, err := getById(shopId)(p.db.WithContext(p.ctx))()
		if err != nil {
			return err
		}
		if err := requireOwner(e, characterId); err != nil {
			return err
		}
		if err := blacklist.NewProcessor(p.l, p.ctx, p.db).Remove(shopId, name); err != nil {
			return err
		}
		return mb.Put(merchant.EnvStatusEventTopic, StatusEventBlacklistUpdatedProvider(characterId, shopId))
	}
}

func (p *ProcessorImpl) GetBlacklist(shopId uuid.UUID) ([]string, error) {
	return blacklist.NewProcessor(p.l, p.ctx, p.db).Names(shopId)
}

func (p *ProcessorImpl) GetVisits(shopId uuid.UUID) ([]visit.Model, error) {
	return visit.NewProcessor(p.l, p.ctx, p.db).List(shopId)
}

func (p *ProcessorImpl) AddToBlacklistAndEmit(shopId uuid.UUID, characterId uint32, name string, bannedCharacterId uint32) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).AddToBlacklist(buf)(shopId, characterId, name, bannedCharacterId)
		})
	})
}

func (p *ProcessorImpl) RemoveFromBlacklistAndEmit(shopId uuid.UUID, characterId uint32, name string) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).RemoveFromBlacklist(buf)(shopId, characterId, name)
		})
	})
}

// ExitShopAndEmit stays on the direct producer path: ExitShop only mutates
// the Redis visitor registry (no Postgres write), so there is no DB state
// change to couple the emit to.
func (p *ProcessorImpl) ExitShopAndEmit(characterId uint32, shopId uuid.UUID) error {
	return message.Emit(p.producer)(func(buf *message.Buffer) error {
		return p.ExitShop(buf)(characterId, shopId)
	})
}

func (p *ProcessorImpl) AddListingAndEmit(shopId uuid.UUID, characterId uint32, itemId uint32, itemType byte, bundleSize uint16, bundleCount uint16, pricePerBundle uint32, itemSnapshot asset2.AssetData, inventoryType byte, assetId uint32) (listing.Model, error) {
	var result listing.Model
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			var err error
			result, err = p.WithTransaction(tx).AddListing(buf)(shopId, characterId, itemId, itemType, bundleSize, bundleCount, pricePerBundle, itemSnapshot, inventoryType, assetId)
			return err
		})
	})
	return result, txErr
}

func (p *ProcessorImpl) RemoveListingAndEmit(shopId uuid.UUID, characterId uint32, listingIndex uint16) (listing.Model, error) {
	var result listing.Model
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			var err error
			result, err = p.WithTransaction(tx).RemoveListing(buf)(shopId, characterId, listingIndex)
			return err
		})
	})
	return result, txErr
}

func (p *ProcessorImpl) PurchaseBundleAndEmit(buyerCharacterId uint32, shopId uuid.UUID, listingIndex uint16, bundleCount uint16, worldId world.Id) (PurchaseResult, error) {
	var result PurchaseResult
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			var err error
			result, err = p.WithTransaction(tx).PurchaseBundle(buf)(buyerCharacterId, shopId, listingIndex, bundleCount, worldId)
			return err
		})
	})
	return result, txErr
}

func (p *ProcessorImpl) SendMessageAndEmit(shopId uuid.UUID, characterId uint32, content string) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).SendMessage(buf)(shopId, characterId, content)
		})
	})
}

func (p *ProcessorImpl) RetrieveFrederickAndEmit(characterId uint32, worldId world.Id) error {
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(buf *message.Buffer) error {
			return p.WithTransaction(tx).RetrieveFrederick(buf)(characterId, worldId)
		})
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
