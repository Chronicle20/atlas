package quest

import (
	"atlas-channel/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "characters/%d/quests"
)

func getBaseRequest() string {
	return requests.RootUrl("QUESTS")
}

func requestByCharacterId(characterId uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, characterId))
}
