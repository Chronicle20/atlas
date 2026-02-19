package wallet

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "accounts/%d/wallet"
)

func getBaseRequest() string {
	return requests.RootUrl("CASHSHOP")
}

func requestByAccountId(accountId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource, accountId))
}
