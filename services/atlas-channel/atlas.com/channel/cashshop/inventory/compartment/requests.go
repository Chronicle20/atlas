package compartment

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "accounts/%d/cash-shop/inventory/compartments"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "CASHSHOP")
}

// requestByAccountId creates a GET request for all compartments for an account
func requestByAccountId(ctx context.Context, accountId uint32) requests.Request[[]RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]RestModel](err)
	}
	return requests.GetRequest[[]RestModel](fmt.Sprintf(root+Resource, accountId))
}

// requestByAccountIdAndType creates a GET request for a specific compartment by account ID and type
func requestByAccountIdAndType(ctx context.Context, accountId uint32, compartmentType CompartmentType) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+Resource+"?type=%d", accountId, byte(compartmentType)))
}
