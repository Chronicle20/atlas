package inventory

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// compartmentByType fetches one inventory compartment (by type) with its assets
// included, mirroring atlas-summons' inventory client.
const compartmentByType = "characters/%d/inventory/compartments?type=%d&include=assets"

var baseURLProvider = func() string {
	return requests.RootUrl("INVENTORY")
}

func getBaseRequest() string {
	return baseURLProvider()
}

func requestCompartmentByType(characterId uint32, inventoryType inventory.Type) requests.Request[CompartmentRestModel] {
	return requests.GetRequest[CompartmentRestModel](fmt.Sprintf(getBaseRequest()+compartmentByType, characterId, inventoryType))
}

// SetBaseURLForTest swaps the base URL for httptest-backed tests. Only call
// from a test; production uses the env-driven RootUrl("INVENTORY") default.
func SetBaseURLForTest(url string) func() {
	prev := baseURLProvider
	baseURLProvider = func() string { return url + "/api/" }
	return func() { baseURLProvider = prev }
}
