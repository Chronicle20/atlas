package item

import (
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	itemInformationEquipmentResource = "equipment"
	itemInformationEquipResource     = itemInformationEquipmentResource + "/%d"
	itemInformationEquipSlotResource = itemInformationEquipResource + "/slots"
)

func getBaseRequest() string {
	return os.Getenv("ITEM_INFORMATION_SERVICE_URL")
}

func requestEquipmentSlotDestination(itemId uint32) requests.Request[attributes] {
	return requests.MakeGetRequest[attributes](fmt.Sprintf(getBaseRequest()+itemInformationEquipSlotResource, itemId))
}
