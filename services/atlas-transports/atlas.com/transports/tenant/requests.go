package tenant

import (
	"atlas-transports/rest"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	tenantsResource = "tenants"
)

func getBaseRequest() string {
	return requests.RootUrl("TENANTS")
}

func requestAll() requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](getBaseRequest() + tenantsResource)
}