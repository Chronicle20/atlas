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
// tenant. atlas-tenants serves this resource as a single JSON:API record
// (`{"data": {...}}`), not an array, so the request is typed for a single
// RpsRewardRestModel.
func requestRewards(tenantId string) requests.Request[RpsRewardRestModel] {
	url := fmt.Sprintf("%stenants/%s/%s/%s", getBaseRequest(), tenantId, configurationsResource, rpsRewardsResource)
	return requests.GetRequest[RpsRewardRestModel](url)
}
