package buddy

import (
	"atlas-query-aggregator/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource      = "buddy-list"
	ByCharacterId = Resource + "/character/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("BUDDIES")
}

func requestByCharacterId(characterId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ByCharacterId, characterId))
}
