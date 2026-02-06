package configuration

import (
	"atlas-character-factory/configuration/tenant"
	"atlas-character-factory/rest"

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
	return rest.MakeGetRequest[[]tenant.RestModel](getBaseRequest() + AllTenants)
}
