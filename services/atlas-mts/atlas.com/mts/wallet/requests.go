package wallet

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Resource is the cash-shop wallet GET path template (accountId-keyed). It matches
// services/atlas-cashshop/.../wallet/resource.go's GET /accounts/{accountId}/wallet.
const Resource = "accounts/%d/wallet"

func getBaseRequest() string {
	return requests.RootUrl("CASHSHOP")
}

func requestByAccountId(accountId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource, accountId))
}

// createRequest POSTs a new cash-shop wallet for the account (JSON:API enveloped
// by the requests layer). Matches cashshop's POST /accounts/{accountId}/wallet
// (handleCreateWallet), which reads accountId from the path and credit/points/
// prepaid from the body.
func createRequest(accountId uint32, rm RestModel) requests.Request[RestModel] {
	return requests.PostRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource, accountId), rm)
}
