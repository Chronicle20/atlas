package tenant

import (
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	tenantsResource = "tenants"
)

func getBaseRequest() string {
	return requests.RootUrl("TENANTS")
}

func requestAll() requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](getBaseRequest() + tenantsResource)
}
