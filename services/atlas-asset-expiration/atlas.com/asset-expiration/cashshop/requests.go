package cashshop

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Inventory    = "accounts/%d/cash-shop/inventory"
	Compartments = "accounts/%d/cash-shop/inventory/compartments"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "CASHSHOP")
}

func requestCompartments(ctx context.Context, accountId uint32) requests.Request[[]CompartmentRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]CompartmentRestModel](err)
	}
	return requests.GetRequest[[]CompartmentRestModel](fmt.Sprintf(root+Compartments, accountId))
}
