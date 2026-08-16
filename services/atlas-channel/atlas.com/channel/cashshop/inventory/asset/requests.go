package asset

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "accounts/%d/cash-shop/inventory/compartments/%s/assets"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "CASHSHOP")
}

// requestById creates a GET request for a specific asset by ID
func requestById(ctx context.Context, accountId uint32, compartmentId uuid.UUID, assetId uint32) requests.Request[RestModel] {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+Resource+"/%d", accountId, compartmentId.String(), assetId))
}

// requestByCompartmentId creates a GET request for all assets in a compartment
func requestByCompartmentId(ctx context.Context, accountId uint32, compartmentId uuid.UUID) requests.Request[[]RestModel]  {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]RestModel](err)
	}
	return requests.GetRequest[[]RestModel](fmt.Sprintf(root+Resource, accountId, compartmentId.String()))
}
