package shop

import (
	"atlas-merchant/listing"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
)

type RestModel struct {
	Id           string              `json:"-"`
	CharacterId  uint32              `json:"characterId"`
	ShopType     byte                `json:"shopType"`
	State        byte                `json:"state"`
	Title        string              `json:"title"`
	WorldId      byte                `json:"worldId"`
	ChannelId    byte                `json:"channelId"`
	MapId        uint32              `json:"mapId"`
	InstanceId   string              `json:"instanceId"`
	X            int16               `json:"x"`
	Y            int16               `json:"y"`
	PermitItemId uint32              `json:"permitItemId"`
	CloseReason  byte                `json:"closeReason"`
	MesoBalance  uint32              `json:"mesoBalance"`
	ListingCount int64               `json:"listingCount"`
	Visitors     []uint32            `json:"visitors,omitempty"`
	Listings     []listing.RestModel `json:"-"`
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r RestModel) GetName() string {
	return "merchants"
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
	for _, l := range r.Listings {
		result = append(result, l)
	}
	return result
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:           m.Id().String(),
		CharacterId:  m.CharacterId(),
		ShopType:     byte(m.ShopType()),
		State:        byte(m.State()),
		Title:        m.Title(),
		WorldId:      byte(m.WorldId()),
		ChannelId:    byte(m.ChannelId()),
		MapId:        m.MapId(),
		InstanceId:   m.InstanceId().String(),
		X:            m.X(),
		Y:            m.Y(),
		PermitItemId: m.PermitItemId(),
		CloseReason:  byte(m.CloseReason()),
		MesoBalance:  m.MesoBalance(),
	}, nil
}

func TransformWithListingCount(counts map[uuid.UUID]int64) func(m Model) (RestModel, error) {
	return func(m Model) (RestModel, error) {
		rm, err := Transform(m)
		if err != nil {
			return RestModel{}, err
		}
		rm.ListingCount = counts[m.Id()]
		return rm, nil
	}
}

func TransformWithListings(listings []listing.Model) func(m Model) (RestModel, error) {
	return func(m Model) (RestModel, error) {
		rm, err := Transform(m)
		if err != nil {
			return RestModel{}, err
		}
		listingRest := make([]listing.RestModel, 0, len(listings))
		for _, l := range listings {
			lr, err := listing.Transform(l)
			if err != nil {
				return RestModel{}, err
			}
			listingRest = append(listingRest, lr)
		}
		rm.Listings = listingRest
		return rm, nil
	}
}

func TransformWithListingsAndVisitors(listings []listing.Model, visitors []uint32) func(m Model) (RestModel, error) {
	return func(m Model) (RestModel, error) {
		rm, err := TransformWithListings(listings)(m)
		if err != nil {
			return RestModel{}, err
		}
		rm.Visitors = visitors
		rm.ListingCount = int64(len(listings))
		return rm, nil
	}
}

type ListingSearchRestModel struct {
	Id               string `json:"-"`
	ShopId           string `json:"shopId"`
	ShopTitle        string `json:"shopTitle"`
	WorldId          byte   `json:"worldId"`
	ChannelId        byte   `json:"channelId"`
	MapId            uint32 `json:"mapId"`
	ItemId           uint32 `json:"itemId"`
	ItemType         byte   `json:"itemType"`
	Quantity         uint16 `json:"quantity"`
	BundleSize       uint16 `json:"bundleSize"`
	BundlesRemaining uint16 `json:"bundlesRemaining"`
	PricePerBundle   uint32 `json:"pricePerBundle"`
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

func TransformSearchResult(sr ListingSearchResult) (ListingSearchRestModel, error) {
	return ListingSearchRestModel{
		Id:               sr.Listing.Id().String(),
		ShopId:           sr.ShopId.String(),
		ShopTitle:        sr.Title,
		WorldId:          byte(sr.WorldId),
		ChannelId:        byte(sr.ChannelId),
		MapId:            sr.MapId,
		ItemId:           sr.Listing.ItemId(),
		ItemType:         sr.Listing.ItemType(),
		Quantity:         sr.Listing.Quantity(),
		BundleSize:       sr.Listing.BundleSize(),
		BundlesRemaining: sr.Listing.BundlesRemaining(),
		PricePerBundle:   sr.Listing.PricePerBundle(),
	}, nil
}

