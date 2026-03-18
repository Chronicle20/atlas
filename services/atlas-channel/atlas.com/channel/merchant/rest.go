package merchant

import (
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
)

type RestModel struct {
	Id           string             `json:"-"`
	CharacterId  uint32             `json:"characterId"`
	ShopType     byte               `json:"shopType"`
	State        byte               `json:"state"`
	Title        string             `json:"title"`
	MapId        uint32             `json:"mapId"`
	X            int16              `json:"x"`
	Y            int16              `json:"y"`
	PermitItemId uint32             `json:"permitItemId"`
	MesoBalance  uint32             `json:"mesoBalance"`
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
		x:            rm.X,
		y:            rm.Y,
		permitItemId: rm.PermitItemId,
		mesoBalance:  rm.MesoBalance,
		listingCount: rm.ListingCount,
		visitors:     rm.Visitors,
		listings:     ls,
	}, nil
}
