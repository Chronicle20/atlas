package quest

import (
	"atlas-quest/data"
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
	return data.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+questPath, questId))
}

func requestAutoStartQuests() requests.Request[[]RestModel] {
	return data.MakeGetRequest[[]RestModel](getBaseRequest() + autoStartQuestsPath)
}
