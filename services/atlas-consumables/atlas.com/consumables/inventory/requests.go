package inventory

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource              = "characters/%d/inventory"
	ById                  = Resource
	accommodationResource = "characters/%d/inventory/accommodation"
)

func getBaseRequest() string {
	return requests.RootUrl("INVENTORY")
}

func requestById(id uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, id))
}

func requestCheckAccommodation(characterId uint32, items []AccommodationRequest) requests.Request[accommodationOutputRestModel] {
	body := accommodationInputRestModel{Id: fmt.Sprintf("%d", characterId)}
	for _, it := range items {
		body.Items = append(body.Items, accommodationItemRestModel{ItemId: it.ItemId, Quantity: it.Quantity})
	}
	return requests.PostRequest[accommodationOutputRestModel](fmt.Sprintf(getBaseRequest()+accommodationResource, characterId), body)
}
