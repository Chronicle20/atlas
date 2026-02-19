package party_quest

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource    = "party-quests/instances"
	ByCharacter = Resource + "/character/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("PARTY_QUESTS")
}

func requestByCharacter(characterId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ByCharacter, characterId))
}
