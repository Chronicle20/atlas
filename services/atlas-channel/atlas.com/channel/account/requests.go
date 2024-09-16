package account

import (
	"atlas-channel/rest"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-tenant"
	"os"
)

const (
	AccountsResource = "accounts"
	AccountsById     = AccountsResource + "/%d"
)

func getBaseRequest() string {
	return os.Getenv("ACCOUNT_SERVICE_URL")
}

func requestAccounts(ctx context.Context, tenant tenant.Model) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest() + AccountsResource))
}

func requestAccountById(ctx context.Context, tenant tenant.Model) func(id uint32) requests.Request[RestModel] {
	return func(id uint32) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+AccountsById, id))
	}
}
