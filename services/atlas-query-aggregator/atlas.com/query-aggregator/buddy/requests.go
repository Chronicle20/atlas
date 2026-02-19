package buddy

import (
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
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ByCharacterId, characterId))
}
