package channel

import (
	"atlas-login/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	WorldsResource = "worlds/"
	WorldsById     = WorldsResource + "%d"
	Resource       = WorldsById + "/channels"
)

func getBaseRequest() string {
	return os.Getenv("WORLD_SERVICE_URL")
}

func requestChannelsForWorld(worldId byte) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, worldId))
}
