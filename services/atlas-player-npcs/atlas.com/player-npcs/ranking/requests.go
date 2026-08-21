package ranking

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "rankings/characters"
	ById     = Resource + "/%d?filter[worldId]=%d"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "RANKINGS")
}

func requestById(ctx context.Context, characterId uint32, worldId world.Id) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+ById, characterId, worldId))
}
