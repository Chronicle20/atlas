package monster

import (
	"atlas-channel/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	mapMonstersResource = "worlds/%d/channels/%d/maps/%d/monsters"
	monstersResource    = "monsters/%d"
)

func getBaseRequest() string {
	return os.Getenv("MONSTER_SERVICE_URL")
}

func requestInMap(worldId byte, channelId byte, mapId uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+mapMonstersResource, worldId, channelId, mapId))
}

func requestById(uniqueId uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+monstersResource, uniqueId))
}
