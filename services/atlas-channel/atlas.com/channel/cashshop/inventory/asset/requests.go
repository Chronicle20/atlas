package asset

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/google/uuid"
)

const (
	Resource = "accounts/%d/cash-shop/inventory/compartments/%s/assets"
)

func getBaseRequest() string {
	return requests.RootUrl("CASHSHOP")
}

// requestById creates a GET request for a specific asset by ID
func requestById(accountId uint32, compartmentId uuid.UUID, assetId uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+Resource+"/%d", accountId, compartmentId.String(), assetId))
}

// requestByCompartmentId creates a GET request for all assets in a compartment
func requestByCompartmentId(accountId uint32, compartmentId uuid.UUID) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+Resource, accountId, compartmentId.String()))
}
