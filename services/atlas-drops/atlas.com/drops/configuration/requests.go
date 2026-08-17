package configuration

import (
	"context"
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
