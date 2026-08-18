package inventory

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// compartmentByType fetches the equip compartment (inventory type 1) with its
// assets included. Mirrors atlas-effective-stats' inventory client.
const compartmentByType = "characters/%d/inventory/compartments?type=%d&include=assets"

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "INVENTORY")
}

func requestEquipCompartment(ctx context.Context, characterId uint32) requests.Request[CompartmentRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[CompartmentRestModel](err)
	}
	return requests.GetRequest[CompartmentRestModel](fmt.Sprintf(root+compartmentByType, characterId, 1))
}
