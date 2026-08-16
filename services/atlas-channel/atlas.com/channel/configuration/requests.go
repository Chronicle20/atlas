package configuration

import (
	"context"
	"atlas-channel/configuration/tenant"
	"fmt"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource  = "configurations"
	ByService = Resource + "/services/%s"
	ForTenant = Resource + "/tenants/%s"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "CONFIGURATIONS")
}

func requestByService(ctx context.Context, serviceId uuid.UUID) requests.Request[RestModel] {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+ByService, serviceId.String()))
}

func requestForTenant(ctx context.Context, tenantId uuid.UUID) requests.Request[tenant.RestModel]  {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[tenant.RestModel](err)
	}
	return requests.GetRequest[tenant.RestModel](fmt.Sprintf(root+ForTenant, tenantId.String()))
}
