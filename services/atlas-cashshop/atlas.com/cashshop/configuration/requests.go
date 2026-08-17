package configuration

import (
	"atlas-cashshop/configuration/tenant"
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource  = "configurations"
	ForTenant = Resource + "/tenants/%s"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "CONFIGURATIONS")
}

func RequestForTenant(ctx context.Context, tenantId uuid.UUID) requests.Request[tenant.RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[tenant.RestModel](err)
	}
	return requests.GetRequest[tenant.RestModel](fmt.Sprintf(root+ForTenant, tenantId.String()))
}
