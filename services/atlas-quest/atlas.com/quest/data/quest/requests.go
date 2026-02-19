package quest

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	questPath           = "data/quests/%d"
	autoStartQuestsPath = "data/quests/auto-start"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestQuestById(questId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+questPath, questId))
}

func requestAutoStartQuests() requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](getBaseRequest() + autoStartQuestsPath)
}
