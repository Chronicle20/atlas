package compartment

import (
	"atlas-channel/rest"
	"fmt"
	"github.com/Chronicle20/atlas-constants/inventory"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "characters/%d/inventory/compartments"
	ByType   = Resource + "?type=%d"
)

func getBaseRequest() string {
	return requests.RootUrl("INVENTORY")
}

func requestByType(characterId uint32, inventoryType inventory.Type) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ByType, characterId, inventoryType))
}
