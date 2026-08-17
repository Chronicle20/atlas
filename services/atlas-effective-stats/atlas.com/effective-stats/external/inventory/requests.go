package inventory

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	CompartmentByType = "characters/%d/inventory/compartments?type=%d&include=assets"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "INVENTORY")
}

// RequestEquipCompartment returns a request to fetch the equip compartment with assets
// type=1 is the equip inventory type
func RequestEquipCompartment(ctx context.Context, characterId uint32) requests.Request[CompartmentRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[CompartmentRestModel](err)
	}
	return requests.GetRequest[CompartmentRestModel](fmt.Sprintf(root+CompartmentByType, characterId, 1))
}
