package inventory

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// compartmentByType fetches the equip compartment (inventory type 1) with its
// assets included. Mirrors atlas-effective-stats' inventory client.
const compartmentByType = "characters/%d/inventory/compartments?type=%d&include=assets"

func getBaseRequest() string {
	return requests.RootUrl("INVENTORY")
}

func requestEquipCompartment(characterId uint32) requests.Request[CompartmentRestModel] {
	return requests.GetRequest[CompartmentRestModel](fmt.Sprintf(getBaseRequest()+compartmentByType, characterId, 1))
}
