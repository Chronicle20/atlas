package quest

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	questsResource     = "data/quests"
	questByIdResource  = "data/quests/%d"
	autoStartQuests    = "data/quests/auto-start"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(questId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+questByIdResource, questId))
}

func requestAll() requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](getBaseRequest() + questsResource)
}

func requestAutoStart() requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](getBaseRequest() + autoStartQuests)
}
