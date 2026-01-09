package pet

import (
	"atlas-query-aggregator/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource        = "pets"
	ByCharacterId   = "/characters/%d/" + Resource
)

func getBaseRequest() string {
	return requests.RootUrl("PETS")
}

func requestByCharacterId(characterId uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+ByCharacterId, characterId))
}
