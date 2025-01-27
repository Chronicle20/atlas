package thread

import (
	"atlas-channel/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	Resource = "guilds/%d/threads"
	ById     = Resource + "/%d"
)

func getBaseRequest() string {
	return os.Getenv("BASE_SERVICE_URL")
}

func requestById(guildId uint32, threadId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, guildId, threadId))
}

func requestAll(guildId uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, guildId))
}
