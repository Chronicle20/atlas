package monster

import (
	"atlas-maps/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	mapsResource     = "maps/"
	monstersResource = mapsResource + "%d/monsters"
)

func getBaseRequest() string {
	return os.Getenv("GAME_DATA_SERVICE_URL")
}

func requestSpawnPoints(mapId uint32) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+monstersResource, mapId))
}
