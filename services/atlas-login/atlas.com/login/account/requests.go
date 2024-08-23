package account

import (
	"atlas-login/rest"
	"atlas-login/tenant"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	AccountsResource = "accounts"
	AccountsByName   = AccountsResource + "?name=%s"
	AccountsById     = AccountsResource + "/%d"
	Update           = AccountsResource + "/%d"
)

func getBaseRequest() string {
	return os.Getenv("ACCOUNT_SERVICE_URL")
}

func requestAccounts(ctx context.Context, tenant tenant.Model) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest() + AccountsResource))
}

func requestAccountByName(ctx context.Context, tenant tenant.Model) func(name string) requests.Request[RestModel] {
	return func(name string) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+AccountsByName, name))
	}
}

func requestAccountById(ctx context.Context, tenant tenant.Model) func(id uint32) requests.Request[RestModel] {
	return func(id uint32) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+AccountsById, id))
	}
}

func requestUpdate(ctx context.Context, tenant tenant.Model) func(m Model) requests.Request[RestModel] {
	return func(m Model) requests.Request[RestModel] {
		im, _ := Transform(m)
		return rest.MakePatchRequest[RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+Update, m.id), im)
	}
}
