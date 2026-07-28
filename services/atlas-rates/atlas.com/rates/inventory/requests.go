package inventory

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource          = "characters/%d/inventory"
	CompartmentAssets = "characters/%d/inventory/compartments/%s/assets"
)

func getBaseRequest() string {
	return requests.RootUrl("INVENTORY")
}

func requestInventory(characterId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource, characterId))
}

// compartmentAssetsUrl returns the list URL for a compartment's assets. It is
// a bare URL (not a requests.Request) because the list is now paginated
// server-side (task-117) and consumed via requests.DrainProvider, which
// appends its own page[number]/page[size] query params per request.
func compartmentAssetsUrl(characterId uint32, compartmentId string) string {
	return fmt.Sprintf(getBaseRequest()+CompartmentAssets, characterId, compartmentId)
}
