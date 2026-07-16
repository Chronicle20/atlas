package inventory

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const accommodationResource = "characters/%d/inventory/accommodation"

// AccommodationRequest is one item to ask atlas-inventory whether it could grant.
type AccommodationRequest struct {
	ItemId   uint32
	Quantity uint32
}

type accommodationInputRestModel struct {
	Id    string                       `json:"-"`
	Items []accommodationItemRestModel `json:"items"`
}

type accommodationItemRestModel struct {
	ItemId   uint32 `json:"itemId"`
	Quantity uint32 `json:"quantity"`
}

func (accommodationInputRestModel) GetName() string                          { return "inventoryAccommodations" }
func (m accommodationInputRestModel) GetID() string                          { return m.Id }
func (m *accommodationInputRestModel) SetID(id string) error                 { m.Id = id; return nil }
func (m *accommodationInputRestModel) SetToOneReferenceID(_, _ string) error { return nil }
func (m *accommodationInputRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

type accommodationOutputRestModel struct {
	Id           string                         `json:"-"`
	Accommodated bool                           `json:"accommodated"`
	Results      []accommodationResultRestModel `json:"results"`
}

type accommodationResultRestModel struct {
	ItemId       uint32 `json:"itemId"`
	Quantity     uint32 `json:"quantity"`
	Accommodated bool   `json:"accommodated"`
}

func (accommodationOutputRestModel) GetName() string                          { return "inventoryAccommodations" }
func (m accommodationOutputRestModel) GetID() string                          { return m.Id }
func (m *accommodationOutputRestModel) SetID(id string) error                 { m.Id = id; return nil }
func (m *accommodationOutputRestModel) SetToOneReferenceID(_, _ string) error { return nil }
func (m *accommodationOutputRestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

func requestCheckAccommodation(characterId uint32, items []AccommodationRequest) requests.Request[accommodationOutputRestModel] {
	body := accommodationInputRestModel{Id: fmt.Sprintf("%d", characterId)}
	for _, it := range items {
		body.Items = append(body.Items, accommodationItemRestModel{ItemId: it.ItemId, Quantity: it.Quantity})
	}
	return requests.PostRequest[accommodationOutputRestModel](fmt.Sprintf(getBaseRequest()+accommodationResource, characterId), body)
}
