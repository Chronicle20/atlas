package item

import (
	"atlas-pets/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "characters"
	BySlot   = Resource + "/%d/inventories/%d/items?slot=%d"
)

func getBaseRequest() string {
	return requests.RootUrl("CHARACTERS")
}

func requestItemBySlot(characterId uint32, inventoryId byte, slot int16) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+BySlot, characterId, inventoryId, slot))
}
