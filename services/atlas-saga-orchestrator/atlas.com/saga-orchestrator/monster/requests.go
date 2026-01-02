package monster

import (
	"atlas-saga-orchestrator/rest"
	"fmt"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	spawnMonsterPath = "worlds/%d/channels/%d/maps/%d/monsters"
)

func getBaseRequest() string {
	return requests.RootUrl("MONSTERS")
}

func requestSpawnMonster(worldId world.Id, channelId channel.Id, mapId uint32, input SpawnInputRestModel) requests.Request[SpawnResponseRestModel] {
	return rest.MakePostRequest[SpawnResponseRestModel](fmt.Sprintf(getBaseRequest()+spawnMonsterPath, worldId, channelId, mapId), input)
}
