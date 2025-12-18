package asset

import (
	"atlas-channel/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/google/uuid"
)

const (
	Resource = "accounts/%d/cash-shop/inventory/compartments/%s/assets"
)

func getBaseRequest() string {
	return requests.RootUrl("CASHSHOP")
}

// requestById creates a GET request for a specific asset by ID
func requestById(accountId uint32, compartmentId uuid.UUID, assetId uuid.UUID) requests.Request[RestModel] {
	return rest.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource+"/%s", accountId, compartmentId.String(), assetId.String()))
}