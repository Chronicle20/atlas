package shop

import (
	"atlas-merchant/listing"

	"github.com/jtumidanski/api2go/jsonapi"
)

type RestModel struct {
	Id           string              `json:"-"`
	CharacterId  uint32              `json:"characterId"`
	ShopType     byte                `json:"shopType"`
	State        byte                `json:"state"`
	Title        string              `json:"title"`
	MapId        uint32              `json:"mapId"`
	X            int16               `json:"x"`
	Y            int16               `json:"y"`
	PermitItemId uint32              `json:"permitItemId"`
	CloseReason  byte                `json:"closeReason"`
	MesoBalance  uint32              `json:"mesoBalance"`
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
		MapId:        m.MapId(),
		X:            m.X(),
		Y:            m.Y(),
		PermitItemId: m.PermitItemId(),
		CloseReason:  byte(m.CloseReason()),
		MesoBalance:  m.MesoBalance(),
	}, nil
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

