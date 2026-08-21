package configuration

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const byTenant = "tenants/%s/configurations/player-npcs"

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "TENANTS")
}

func requestByTenantId(ctx context.Context, tenantId uuid.UUID) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+byTenant, tenantId))
}
