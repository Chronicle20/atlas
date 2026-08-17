package configuration

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	configurationsResource = "configurations"
	rpsRewardsResource     = "rps-rewards"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "TENANTS")
}

// requestRewards creates a request for the rps-rewards configuration for a
// tenant. atlas-tenants serves configuration resources uniformly as a JSON:API
// collection (`{"data": [{...}]}`), so the request is typed for a slice of
// RpsRewardRestModel.
func requestRewards(ctx context.Context, tenantId string) requests.Request[[]RpsRewardRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]RpsRewardRestModel](err)
	}
	url := fmt.Sprintf("%stenants/%s/%s/%s", root, tenantId, configurationsResource, rpsRewardsResource)
	return requests.GetRequest[[]RpsRewardRestModel](url)
}
