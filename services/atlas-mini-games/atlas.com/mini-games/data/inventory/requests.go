package inventory

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// compartmentByType fetches one inventory compartment (by type) with its assets
// included, mirroring atlas-summons' inventory client.
const compartmentByType = "characters/%d/inventory/compartments?type=%d&include=assets"

func getBaseRequest() string {
	return requests.RootUrl("INVENTORY")
}

func requestCompartmentByType(characterId uint32, inventoryType inventory.Type) requests.Request[CompartmentRestModel] {
	return requests.GetRequest[CompartmentRestModel](fmt.Sprintf(getBaseRequest()+compartmentByType, characterId, inventoryType))
}
