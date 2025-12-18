package account

import (
	"atlas-login/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	AccountsResource = "accounts"
	AccountsByName   = AccountsResource + "?name=%s"
	AccountsById     = AccountsResource + "/%d"
	Update           = AccountsResource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("ACCOUNTS")
}

func requestAccounts() requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](getBaseRequest() + AccountsResource)
}

func requestAccountByName(name string) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+AccountsByName, name))
}

func requestAccountById(id uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+AccountsById, id))
}

func requestUpdate(m Model) requests.Request[RestModel] {
	im, _ := Transform(m)
	return rest.MakePatchRequest[RestModel](fmt.Sprintf(getBaseRequest()+Update, m.id), im)
}
