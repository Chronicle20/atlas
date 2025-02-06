package information

import (
	"atlas-character/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	itemInformationResource = "equipment/"
	itemInformationById     = itemInformationResource + "%d"
	slotsForEquipment       = itemInformationById + "/slots"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestEquipmentSlotDestination(id uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+slotsForEquipment, id))
}
