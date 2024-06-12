package account

import (
	"atlas-login/rest/requests"
	"fmt"
	"os"
)

const (
	AccountsResource = "accounts/"
	AccountsByName   = AccountsResource + "?name=%s"
	AccountsById     = AccountsResource + "%d"
)

func getBaseRequest() string {
	return os.Getenv("ACCOUNT_SERVICE_URL")
}

func requestAccountByName(name string) requests.Request[RestModel] {
	return requests.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+AccountsByName, name))
}

func requestAccountById(id uint32) requests.Request[RestModel] {
	return requests.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+AccountsById, id))
}
