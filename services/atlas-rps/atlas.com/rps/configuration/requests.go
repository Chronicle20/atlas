package configuration

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	configurationsResource = "configurations"
	rpsRewardsResource     = "rps-rewards"
)

func getBaseRequest() string {
	return requests.RootUrl("TENANTS")
}

// requestRewards creates a request for the rps-rewards configuration for a
// tenant. atlas-tenants serves configuration resources uniformly as a JSON:API
// collection (`{"data": [{...}]}`), so the request is typed for a slice of
// RpsRewardRestModel.
func requestRewards(tenantId string) requests.Request[[]RpsRewardRestModel] {
	url := fmt.Sprintf("%stenants/%s/%s/%s", getBaseRequest(), tenantId, configurationsResource, rpsRewardsResource)
	return requests.GetRequest[[]RpsRewardRestModel](url)
}
