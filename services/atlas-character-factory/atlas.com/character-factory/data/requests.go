package data

import (
	"atlas-character-factory/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	itemInformationResource        = "equipment/"
	itemInformationById            = itemInformationResource + "%d"
	itemDestinationSlotInformation = itemInformationById + "/slots"
)

func getBaseRequest() string {
	return os.Getenv("GAME_DATA_SERVICE_URL")
}

func requestById(id uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+itemDestinationSlotInformation, id))
}
