package world

import (
	"atlas-channel/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	WorldsResource = "worlds/"
	WorldsById     = WorldsResource + "%d"
)

func getBaseRequest() string {
	return requests.RootUrl("WORLDS")
}

func requestWorld(worldId byte) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+WorldsById, worldId))
}
