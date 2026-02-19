package compartment

import (
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Resource = "accounts/%d/cash-shop/inventory/compartments"
)

func getBaseRequest() string {
	return requests.RootUrl("CASHSHOP")
}

// requestByAccountId creates a GET request for all compartments for an account
func requestByAccountId(accountId uint32) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, accountId))
}

// requestByAccountIdAndType creates a GET request for a specific compartment by account ID and type
func requestByAccountIdAndType(accountId uint32, compartmentType CompartmentType) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource+"?type=%d", accountId, byte(compartmentType)))
}
