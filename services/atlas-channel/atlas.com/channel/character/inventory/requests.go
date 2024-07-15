package inventory

import (
	"os"
)

const (
	charactersResource          = "characters/"
	charactersInventoryResource = charactersResource + "%d/inventories/"
	characterItems              = charactersInventoryResource + "?type=%s&include=inventoryItems,equipmentStatistics"
)

func getBaseRequest() string {
	return os.Getenv("CHARACTER_SERVICE_URL")
}

//func requestEquippedItemsForCharacter(characterId uint32) requests.Request[inventoryAttributes] {
//	return requestItemsForCharacter(characterId, "equip")
//}
//
//func requestItemsForCharacter(characterId uint32, inventoryType string) requests.Request[inventoryAttributes] {
//	return requests.MakeGetRequest[inventoryAttributes](fmt.Sprintf(getBaseRequest()+characterItems, characterId, inventoryType), requests.AddMappers(equipmentIncludes))
//}
