package thread

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
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

func requestAll(guildId uint32) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, guildId))
}
