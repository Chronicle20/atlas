package inventory

import (
	"atlas-effective-stats/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	CompartmentByType = "characters/%d/inventory/compartments?type=%d&include=assets"
)

func getBaseRequest() string {
	return requests.RootUrl("INVENTORY")
}

// RequestEquipCompartment returns a request to fetch the equip compartment with assets
// type=1 is the equip inventory type
func RequestEquipCompartment(characterId uint32) requests.Request[CompartmentRestModel] {
	return rest.MakeGetRequest[CompartmentRestModel](fmt.Sprintf(getBaseRequest()+CompartmentByType, characterId, 1))
}
