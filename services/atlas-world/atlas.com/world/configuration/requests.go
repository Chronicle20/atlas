package configuration

import (
	"atlas-world/configuration/tenant"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource   = "configurations"
	AllTenants = Resource + "/tenants"
)

func getBaseRequest() string {
	return requests.RootUrl("CONFIGURATIONS")
}

func requestAllTenants() requests.Request[[]tenant.RestModel] {
	return requests.GetRequest[[]tenant.RestModel](getBaseRequest() + AllTenants)
}
