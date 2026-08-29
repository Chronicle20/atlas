package compartment

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource              = "characters/%d/inventory/compartments"
	ByType                = Resource + "?type=%d"
	accommodationResource = "characters/%d/inventory/accommodation"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "INVENTORY")
}

func requestByType(ctx context.Context, characterId uint32, inventoryType inventory.Type) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+ByType, characterId, inventoryType))
}

// requestCheckAccommodation asks atlas-inventory whether characterId would
// currently be able to receive every item in items — a list, not one item
// per call, so a recipe's full award set (base output + any random rewards)
// is checked in one round trip (design §4.2.2 step 6).
func requestCheckAccommodation(ctx context.Context, characterId uint32, items []AccommodationItem) requests.Request[accommodationOutputRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[accommodationOutputRestModel](err)
	}
	ris := make([]accommodationItemRestModel, 0, len(items))
	for _, it := range items {
		ris = append(ris, accommodationItemRestModel{ItemId: uint32(it.ItemId), Quantity: it.Quantity})
	}
	body := accommodationInputRestModel{
		Id:    fmt.Sprintf("%d", characterId),
		Items: ris,
	}
	return requests.PostRequest[accommodationOutputRestModel](fmt.Sprintf(root+accommodationResource, characterId), body)
}
