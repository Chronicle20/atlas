package account

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	AccountsResource = "accounts"
	AccountsById     = AccountsResource + "/%d"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "ACCOUNTS")
}

func requestAccountById(ctx context.Context, id uint32) requests.Request[RestModel] {

	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+AccountsById, id))
}
