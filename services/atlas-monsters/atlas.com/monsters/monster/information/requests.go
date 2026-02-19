package information

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	monstersResource = "data/monsters"
	monsterResource  = monstersResource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(monsterId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+monsterResource, monsterId))
}
