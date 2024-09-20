package inventory

import (
	"atlas-messages/character/inventory/item"
	"atlas-messages/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"math"
	"os"
)

const (
	resource               = "characters"
	characterResource      = resource + "/%d"
	characterItemsResource = characterResource + "/inventories/%d/items"
)

func getBaseRequest() string {
	return os.Getenv("CHARACTER_SERVICE_URL")
}

func requestCreateItem(characterId uint32, itemId uint32, quantity uint16) requests.Request[item.RestModel] {
	inventoryType := uint32(math.Floor(float64(itemId) / 1000000))
	i := item.RestModel{ItemId: itemId, Quantity: uint32(quantity)}
	return rest.MakePostRequest[item.RestModel](fmt.Sprintf(getBaseRequest()+characterItemsResource, characterId, inventoryType), i)
}
