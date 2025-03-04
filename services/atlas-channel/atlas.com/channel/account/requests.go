package account

import (
	"atlas-channel/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	AccountsResource = "accounts"
	AccountsById     = AccountsResource + "/%d"
)

func getBaseRequest() string {
	return requests.RootUrl("ACCOUNTS")
}

var requestAccounts = rest.MakeGetRequest[[]RestModel](getBaseRequest() + AccountsResource)

func requestAccountById(id uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+AccountsById, id))
}
