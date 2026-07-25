package setup

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	getSetup = "data/setups/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestSetup(itemId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+getSetup, itemId))
}
