package rates

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	ratesPath = "worlds/%d/channels/%d/characters/%d/rates"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "RATES")
}

func requestRates(ctx context.Context, ch channel.Model, characterId uint32) requests.Request[DataContainer] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[DataContainer](err)
	}
	return requests.GetRequest[DataContainer](fmt.Sprintf(root+ratesPath, ch.WorldId(), ch.Id(), characterId))
}
