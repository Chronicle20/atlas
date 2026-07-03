package incubator

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	configurationsResource = "configurations"
	rewardsResource        = "incubator-rewards"
)

func getBaseRequest() string {
	return requests.RootUrl("TENANTS")
}

// requestRewards creates a request for the incubator-rewards configuration
// resource for a tenant.
func requestRewards(tenantId string) requests.Request[[]RewardRestModel] {
	url := fmt.Sprintf("%stenants/%s/%s/%s", getBaseRequest(), tenantId, configurationsResource, rewardsResource)
	return requests.GetRequest[[]RewardRestModel](url)
}
