package state

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	StartedQuestsResource = "characters/%d/quests/started"
)

func getBaseRequest() string {
	return requests.RootUrl("QUESTS")
}

func requestStartedQuests(characterId uint32) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+StartedQuestsResource, characterId))
}
