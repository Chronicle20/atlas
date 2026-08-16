package inventory

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource              = "characters/%d/inventory"
	ById                  = Resource
	accommodationResource = "characters/%d/inventory/accommodation"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "INVENTORY")
}

func requestById(ctx context.Context, id uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+ById, id))
}

func requestCheckAccommodation(ctx context.Context, characterId uint32, items []AccommodationRequest) requests.Request[accommodationOutputRestModel] {
	body := accommodationInputRestModel{Id: fmt.Sprintf("%d", characterId)}
	for _, it := range items {
		body.Items = append(body.Items, accommodationItemRestModel{ItemId: it.ItemId, Quantity: it.Quantity})
	}
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[accommodationOutputRestModel](err)
	}
	return requests.PostRequest[accommodationOutputRestModel](fmt.Sprintf(root+accommodationResource, characterId), body)
}
