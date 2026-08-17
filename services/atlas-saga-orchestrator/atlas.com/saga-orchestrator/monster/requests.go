package monster

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	spawnMonsterPath = "worlds/%d/channels/%d/maps/%d/instances/%s/monsters"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MONSTERS")
}

func requestSpawnMonster(ctx context.Context, f field.Model, input SpawnInputRestModel) requests.Request[SpawnResponseRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[SpawnResponseRestModel](err)
	}
	return requests.PostRequest[SpawnResponseRestModel](fmt.Sprintf(root+spawnMonsterPath, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()), input)
}
