package npc_spawn

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	spawnNpcPath = "worlds/%d/channels/%d/maps/%d/instances/%s/npcs"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MAPS")
}

func requestSpawnNpc(ctx context.Context, f field.Model, input SpawnInputRestModel) requests.Request[SpawnResponseRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[SpawnResponseRestModel](err)
	}
	return requests.PostRequest[SpawnResponseRestModel](fmt.Sprintf(root+spawnNpcPath, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()), input)
}
