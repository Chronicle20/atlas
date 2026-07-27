package quest

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	StartedQuestsResource = "characters/%d/quests/started"
)

func getBaseRequest() string {
	return requests.RootUrl("QUESTS")
}

// startedQuestsUrl returns the list URL for a character's started quests.
// It is a bare URL (not a requests.Request) because the list is now
// paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func startedQuestsUrl(characterId uint32) string {
	return fmt.Sprintf(getBaseRequest()+StartedQuestsResource, characterId)
}
