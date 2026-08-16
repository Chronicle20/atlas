package rate

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "worlds/%d/channels/%d/characters/%d/rates"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "RATES")
}

func requestByCharacter(ctx context.Context, ch channel.Model, characterId uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+Resource, ch.WorldId(), ch.Id(), characterId))
}
