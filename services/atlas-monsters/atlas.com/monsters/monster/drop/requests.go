package drop

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	monsterDropsResource = "monsters/%d/drops"
)

func getBaseRequest() string {
	return requests.RootUrl("DROPS_INFORMATION")
}

func requestForMonster(monsterId uint32) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+monsterDropsResource, monsterId))
}
