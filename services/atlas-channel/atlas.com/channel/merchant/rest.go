package merchant

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
)

type RestModel struct {
	Id           string             `json:"-"`
	CharacterId  uint32             `json:"characterId"`
	ShopType     byte               `json:"shopType"`
	State        byte               `json:"state"`
	Title        string             `json:"title"`
	WorldId      byte               `json:"worldId"`
	ChannelId    byte               `json:"channelId"`
	MapId        uint32             `json:"mapId"`
	InstanceId   string             `json:"instanceId"`
	X            int16              `json:"x"`
	Y            int16              `json:"y"`
	PermitItemId uint32             `json:"permitItemId"`
	MesoBalance  uint32             `json:"mesoBalance"`
	CreatedAt    time.Time          `json:"createdAt"`
	ListingCount int64              `json:"listingCount"`
	Visitors     []uint32           `json:"visitors,omitempty"`
	Listings     []ListingRestModel `json:"-"`
}

func (r RestModel) GetName() string {
	return "merchants"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "listings",
			Name: "listings",
		},
	}
}

func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	for _, l := range r.Listings {
		result = append(result, jsonapi.ReferenceID{
			ID:   l.GetID(),
			Type: "listings",
			Name: "listings",
		})
	}
	return result
}

func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	for key := range r.Listings {
		result = append(result, r.Listings[key])
	}
	return result
}

func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "listings" {
		for _, id := range IDs {
			r.Listings = append(r.Listings, ListingRestModel{Id: id})
		}
	}
	return nil
}

func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["listings"]; ok {
		listings := make([]ListingRestModel, 0)
		for _, ri := range r.Listings {
			if ref, ok := refMap[ri.GetID()]; ok {
				wip := ri
				err := jsonapi.ProcessIncludeData(&wip, ref, references)
				if err != nil {
					return err
				}
				listings = append(listings, wip)
			}
		}
		r.Listings = listings
	}
	return nil
}

func Extract(rm RestModel) (Model, error) {
	id, err := uuid.Parse(rm.Id)
	if err != nil {
		return Model{}, err
	}

	instanceId, _ := uuid.Parse(rm.InstanceId)

	ls, err := model.SliceMap(ExtractListing)(model.FixedProvider(rm.Listings))(model.ParallelMap())()
	if err != nil {
		return Model{}, err
	}

	return Model{
		id:           id,
		characterId:  rm.CharacterId,
		shopType:     rm.ShopType,
		state:        rm.State,
		title:        rm.Title,
		worldId:      world.Id(rm.WorldId),
		channelId:    channel.Id(rm.ChannelId),
		mapId:        rm.MapId,
		instanceId:   instanceId,
		x:            rm.X,
		y:            rm.Y,
		permitItemId: rm.PermitItemId,
		mesoBalance:  rm.MesoBalance,
		createdAt:    rm.CreatedAt,
		listingCount: rm.ListingCount,
		visitors:     rm.Visitors,
		listings:     ls,
	}, nil
}

// FrederickStatusRestModel mirrors atlas-merchant's frederick.StatusRestModel:
// whether the character has unclaimed items/mesos waiting at Fredrick. The
// entrusted-shop permit check consults it before allowing a new hired merchant.
type FrederickStatusRestModel struct {
	Id         string `json:"-"`
	HasPending bool   `json:"hasPending"`
}

func (r FrederickStatusRestModel) GetName() string {
	return "frederick-status"
}

func (r FrederickStatusRestModel) GetID() string {
	return r.Id
}

func (r *FrederickStatusRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r *FrederickStatusRestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *FrederickStatusRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// SnapshotRestModel mirrors atlas-merchant's asset.AssetData JSON shape —
// the listing's point-in-sale item snapshot, needed to encode the
// GW_ItemSlotBase block for equip rows in the shop-scanner result.
type SnapshotRestModel struct {
	Expiration     time.Time  `json:"expiration"`
	CreatedAt      time.Time  `json:"createdAt"`
	Quantity       uint32     `json:"quantity"`
	OwnerId        uint32     `json:"ownerId"`
	Flag           uint16     `json:"flag"`
	Rechargeable   uint64     `json:"rechargeable"`
	Strength       uint16     `json:"strength"`
	Dexterity      uint16     `json:"dexterity"`
	Intelligence   uint16     `json:"intelligence"`
	Luck           uint16     `json:"luck"`
	Hp             uint16     `json:"hp"`
	Mp             uint16     `json:"mp"`
	WeaponAttack   uint16     `json:"weaponAttack"`
	MagicAttack    uint16     `json:"magicAttack"`
	WeaponDefense  uint16     `json:"weaponDefense"`
	MagicDefense   uint16     `json:"magicDefense"`
	Accuracy       uint16     `json:"accuracy"`
	Avoidability   uint16     `json:"avoidability"`
	Hands          uint16     `json:"hands"`
	Speed          uint16     `json:"speed"`
	Jump           uint16     `json:"jump"`
	Slots          uint16     `json:"slots"`
	LevelType      byte       `json:"levelType"`
	Level          byte       `json:"level"`
	Experience     uint32     `json:"experience"`
	HammersApplied uint32     `json:"hammersApplied"`
	EquippedSince  *time.Time `json:"equippedSince"`
	CashId         int64      `json:"cashId,string"`
	CommodityId    uint32     `json:"commodityId"`
	PurchaseBy     uint32     `json:"purchaseBy"`
	PetId          uint32     `json:"petId"`
}

type ListingSearchRestModel struct {
	Id               string            `json:"-"`
	ShopId           string            `json:"shopId"`
	ShopTitle        string            `json:"shopTitle"`
	WorldId          byte              `json:"worldId"`
	ChannelId        byte              `json:"channelId"`
	MapId            uint32            `json:"mapId"`
	OwnerId          uint32            `json:"ownerId"`
	ShopType         byte              `json:"shopType"`
	State            byte              `json:"state"`
	ItemId           uint32            `json:"itemId"`
	ItemType         byte              `json:"itemType"`
	Quantity         uint16            `json:"quantity"`
	BundleSize       uint16            `json:"bundleSize"`
	BundlesRemaining uint16            `json:"bundlesRemaining"`
	PricePerBundle   uint32            `json:"pricePerBundle"`
	ItemSnapshot     SnapshotRestModel `json:"itemSnapshot"`
}

func (r ListingSearchRestModel) GetID() string {
	return r.Id
}

func (r *ListingSearchRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r ListingSearchRestModel) GetName() string {
	return "listing-search-results"
}

type TopSearchRestModel struct {
	Id     string `json:"-"`
	ItemId uint32 `json:"itemId"`
	Count  uint64 `json:"count"`
}

func (r TopSearchRestModel) GetID() string {
	return r.Id
}

func (r *TopSearchRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r TopSearchRestModel) GetName() string {
	return "shop-search-counts"
}
