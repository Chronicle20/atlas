package thread

import (
	"atlas-channel/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "guilds/%d/threads"
	ById     = Resource + "/%d"
)

func getBaseRequest() string {
	return "http://127.0.0.1:8080/api/"
	//return os.Getenv("BASE_SERVICE_URL")
}

func requestById(guildId uint32, threadId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, guildId, threadId))
}

func requestAll(guildId uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, guildId))
}
