package quest

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
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

// autoStartQuestsUrl is a bare URL (not a requests.Request) because the
// list is now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func autoStartQuestsUrl() string {
	return getBaseRequest() + autoStartQuestsPath
}
