package wallet

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Resource is the cash-shop wallet GET path template (accountId-keyed). It matches
// services/atlas-cashshop/.../wallet/resource.go's GET /accounts/{accountId}/wallet.
const Resource = "accounts/%d/wallet"

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "CASHSHOP")
}

func requestByAccountId(ctx context.Context, accountId uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+Resource, accountId))
}

// createRequest POSTs a new cash-shop wallet for the account (JSON:API enveloped
// by the requests layer). Matches cashshop's POST /accounts/{accountId}/wallet
// (handleCreateWallet), which reads accountId from the path and credit/points/
// prepaid from the body.
func createRequest(ctx context.Context, accountId uint32, rm RestModel) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.PostRequest[RestModel](fmt.Sprintf(root+Resource, accountId), rm)
}
