package party_quest

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

func getBaseRequest() string {
	return requests.RootUrl("PARTY_QUESTS")
}

func requestInstanceByCharacterId(characterId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+"party-quests/instances/character/%d", characterId))
}
