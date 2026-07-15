package thread

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "guilds/%d/threads"
	ById     = Resource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("GUILD_THREADS")
}

func requestById(guildId uint32, threadId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, guildId, threadId))
}

// allUrl returns the list URL for a guild's threads. Bare URL (not a
// requests.Request) because the list is now paginated server-side
// (task-117) and consumed via requests.DrainProvider, which appends its own
// page[number]/page[size] query params per request.
func allUrl(guildId uint32) string {
	return fmt.Sprintf(getBaseRequest()+Resource, guildId)
}
