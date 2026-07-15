package quest

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	questsResource    = "data/quests"
	questByIdResource = "data/quests/%d"
	autoStartQuests   = "data/quests/auto-start"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestById(questId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+questByIdResource, questId))
}

// allQuestsUrl and autoStartQuestsUrl are bare URLs (not requests.Request)
// because both lists are now paginated server-side (task-117) and consumed
// via requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func allQuestsUrl() string {
	return getBaseRequest() + questsResource
}

func autoStartQuestsUrl() string {
	return getBaseRequest() + autoStartQuests
}
