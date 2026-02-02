package inventory

import (
	"atlas-rates/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource          = "characters/%d/inventory"
	CompartmentAssets = "characters/%d/inventory/compartments/%s/assets"
)

func getBaseRequest() string {
	return requests.RootUrl("INVENTORY")
}

func requestInventory(characterId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource, characterId))
}

func requestAssets(characterId uint32, compartmentId string) requests.Request[[]AssetRestModel] {
	return rest.MakeGetRequest[[]AssetRestModel](fmt.Sprintf(getBaseRequest()+CompartmentAssets, characterId, compartmentId))
}
