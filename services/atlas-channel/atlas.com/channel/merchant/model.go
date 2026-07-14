package merchant

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	merchantconst "github.com/Chronicle20/atlas/libs/atlas-constants/merchant"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

type Model struct {
	id           uuid.UUID
	characterId  uint32
	shopType     byte
	state        byte
	title        string
	worldId      world.Id
	channelId    channel.Id
	mapId        uint32
	instanceId   uuid.UUID
	x            int16
	y            int16
	permitItemId uint32
	mesoBalance  uint32
	createdAt    time.Time
	listingCount int64
	visitors     []uint32
	listings     []ListingModel
}

func (m Model) Id() uuid.UUID            { return m.id }
func (m Model) CharacterId() uint32       { return m.characterId }
func (m Model) ShopType() byte            { return m.shopType }
func (m Model) State() byte               { return m.state }
func (m Model) Title() string             { return m.title }
func (m Model) WorldId() world.Id         { return m.worldId }
func (m Model) ChannelId() channel.Id     { return m.channelId }
func (m Model) MapId() uint32             { return m.mapId }
func (m Model) InstanceId() uuid.UUID     { return m.instanceId }
func (m Model) X() int16                  { return m.x }
func (m Model) Y() int16                  { return m.y }
func (m Model) PermitItemId() uint32      { return m.permitItemId }
func (m Model) MesoBalance() uint32       { return m.mesoBalance }
func (m Model) CreatedAt() time.Time      { return m.createdAt }
func (m Model) ListingCount() int64       { return m.listingCount }
func (m Model) Visitors() []uint32        { return m.visitors }
func (m Model) Listings() []ListingModel  { return m.listings }

// Shop states derived from the shared atlas-constants enum (the same source
// atlas-merchant persists), exposed as byte to match the wire model.
const (
	StateDraft       = byte(merchantconst.ShopStateDraft)
	StateOpen        = byte(merchantconst.ShopStateOpen)
	StateMaintenance = byte(merchantconst.ShopStateMaintenance)
	StateClosed      = byte(merchantconst.ShopStateClosed)
)

type SearchListing struct {
	shopId           uuid.UUID
	title            string
	worldId          world.Id
	channelId        channel.Id
	mapId            uint32
	ownerId          uint32
	shopType         byte
	state            byte
	itemId           uint32
	itemType         byte
	quantity         uint16
	bundleSize       uint16
	bundlesRemaining uint16
	pricePerBundle   uint32
	itemSnapshot     SnapshotRestModel
}

func (m SearchListing) ShopId() uuid.UUID               { return m.shopId }
func (m SearchListing) Title() string                   { return m.title }
func (m SearchListing) WorldId() world.Id               { return m.worldId }
func (m SearchListing) ChannelId() channel.Id           { return m.channelId }
func (m SearchListing) MapId() uint32                   { return m.mapId }
func (m SearchListing) OwnerId() uint32                 { return m.ownerId }
func (m SearchListing) ShopType() byte                  { return m.shopType }
func (m SearchListing) State() byte                     { return m.state }
func (m SearchListing) ItemId() uint32                  { return m.itemId }
func (m SearchListing) ItemType() byte                  { return m.itemType }
func (m SearchListing) Quantity() uint16                { return m.quantity }
func (m SearchListing) BundleSize() uint16              { return m.bundleSize }
func (m SearchListing) BundlesRemaining() uint16        { return m.bundlesRemaining }
func (m SearchListing) PricePerBundle() uint32          { return m.pricePerBundle }
func (m SearchListing) ItemSnapshot() SnapshotRestModel { return m.itemSnapshot }

// SearchListingSeed carries the constructor arguments for SearchListing.
type SearchListingSeed struct {
	ShopId           uuid.UUID
	Title            string
	WorldId          world.Id
	ChannelId        channel.Id
	MapId            uint32
	OwnerId          uint32
	ShopType         byte
	State            byte
	ItemId           uint32
	ItemType         byte
	Quantity         uint16
	BundleSize       uint16
	BundlesRemaining uint16
	PricePerBundle   uint32
	Snapshot         SnapshotRestModel
}

// NewSearchListing builds a SearchListing from explicit values (the model's
// constructor for locally-built values; Extract remains the REST path).
func NewSearchListing(s SearchListingSeed) SearchListing {
	return SearchListing{
		shopId:           s.ShopId,
		title:            s.Title,
		worldId:          s.WorldId,
		channelId:        s.ChannelId,
		mapId:            s.MapId,
		ownerId:          s.OwnerId,
		shopType:         s.ShopType,
		state:            s.State,
		itemId:           s.ItemId,
		itemType:         s.ItemType,
		quantity:         s.Quantity,
		bundleSize:       s.BundleSize,
		bundlesRemaining: s.BundlesRemaining,
		pricePerBundle:   s.PricePerBundle,
		itemSnapshot:     s.Snapshot,
	}
}

func ExtractSearchListing(rm ListingSearchRestModel) (SearchListing, error) {
	shopId, err := uuid.Parse(rm.ShopId)
	if err != nil {
		return SearchListing{}, err
	}
	return SearchListing{
		shopId:           shopId,
		title:            rm.ShopTitle,
		worldId:          world.Id(rm.WorldId),
		channelId:        channel.Id(rm.ChannelId),
		mapId:            rm.MapId,
		ownerId:          rm.OwnerId,
		shopType:         rm.ShopType,
		state:            rm.State,
		itemId:           rm.ItemId,
		itemType:         rm.ItemType,
		quantity:         rm.Quantity,
		bundleSize:       rm.BundleSize,
		bundlesRemaining: rm.BundlesRemaining,
		pricePerBundle:   rm.PricePerBundle,
		itemSnapshot:     rm.ItemSnapshot,
	}, nil
}

type TopSearch struct {
	itemId uint32
	count  uint64
}

func (m TopSearch) ItemId() uint32 { return m.itemId }
func (m TopSearch) Count() uint64  { return m.count }

func ExtractTopSearch(rm TopSearchRestModel) (TopSearch, error) {
	return TopSearch{itemId: rm.ItemId, count: rm.Count}, nil
}
