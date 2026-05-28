package monster

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	monstersResource = "data/monsters/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(monsterId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+monstersResource, monsterId))
}
