package slot

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	itemInformationResource = "data/equipment/"
	itemInformationById     = itemInformationResource + "%d"
	slotsForEquipment       = itemInformationById + "/slots"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestEquipmentSlotDestination(id uint32) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+slotsForEquipment, id))
}
