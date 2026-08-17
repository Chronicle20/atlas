package effectivestats

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const statsResource = "worlds/%d/channels/%d/characters/%d/stats"

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "EFFECTIVE_STATS")
}

func requestByCharacter(ctx context.Context, worldId world.Id, channelId channel.Id, characterId uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+statsResource, worldId, channelId, characterId))
}
