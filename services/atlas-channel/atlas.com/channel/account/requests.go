package account

import (
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

var requestAccounts = requests.GetRequest[[]RestModel](getBaseRequest() + AccountsResource)

func requestAccountById(id uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+AccountsById, id))
}
