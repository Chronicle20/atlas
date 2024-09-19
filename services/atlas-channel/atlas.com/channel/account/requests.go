package account

import (
	"atlas-channel/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	AccountsResource = "accounts"
	AccountsById     = AccountsResource + "/%d"
)

func getBaseRequest() string {
	return os.Getenv("ACCOUNT_SERVICE_URL")
}

var requestAccounts = rest.MakeGetRequest[[]RestModel](fmt.Sprintf(getBaseRequest() + AccountsResource))

func requestAccountById(id uint32) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+AccountsById, id))
}
