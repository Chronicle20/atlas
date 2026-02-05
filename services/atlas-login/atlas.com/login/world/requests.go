package world

import (
	"atlas-login/rest"
	"fmt"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	WorldsResource        = "worlds"
	WorldsIncludeChannels = WorldsResource + "?include=channels"
	WorldsById            = WorldsResource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("WORLDS")
}

func requestWorlds() requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](getBaseRequest() + WorldsIncludeChannels)
}

func requestWorld(worldId world.Id) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+WorldsById, worldId))
}
