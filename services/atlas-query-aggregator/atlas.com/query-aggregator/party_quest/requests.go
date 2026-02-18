package party_quest

import (
	"atlas-query-aggregator/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

func getBaseRequest() string {
	return requests.RootUrl("PARTY_QUESTS")
}

func requestInstanceByCharacterId(characterId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+"party-quests/instances/character/%d", characterId))
}
