package setup

import (
	"atlas-npc/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	itemInformationResource = "data/setups/"
	itemInformationById     = itemInformationResource + "%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(id uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+itemInformationById, id))
}
