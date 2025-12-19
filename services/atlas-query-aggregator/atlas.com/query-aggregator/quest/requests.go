package quest

import (
	"atlas-query-aggregator/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "quests"
	ById     = Resource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("QUESTS")
}

func requestById(characterId uint32, questId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById+"?characterId=%d", questId, characterId))
}