package marriage

import (
	"atlas-query-aggregator/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "marriage"
	ByCharacterId = Resource + "/character/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("MARRIAGE")
}

func requestByCharacterId(characterId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ByCharacterId, characterId))
}