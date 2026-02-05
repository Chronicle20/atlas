package monster

import (
	"fmt"

	"atlas-saga-orchestrator/rest"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	spawnMonsterPath = "worlds/%d/channels/%d/maps/%d/instances/%s/monsters"
)

func getBaseRequest() string {
	return requests.RootUrl("MONSTERS")
}

func requestSpawnMonster(f field.Model, input SpawnInputRestModel) requests.Request[SpawnResponseRestModel] {
	return rest.MakePostRequest[SpawnResponseRestModel](fmt.Sprintf(getBaseRequest()+spawnMonsterPath, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()), input)
}
