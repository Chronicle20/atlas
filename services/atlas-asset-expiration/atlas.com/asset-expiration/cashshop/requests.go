package cashshop

import (
	"atlas-asset-expiration/rest"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	Inventory    = "accounts/%d/cash-shop/inventory"
	Compartments = "accounts/%d/cash-shop/inventory/compartments"
)

func getBaseRequest() string {
	return requests.RootUrl("CASHSHOP")
}

func requestInventory(accountId uint32) requests.Request[InventoryRestModel] {
	return rest.MakeGetRequest[InventoryRestModel](fmt.Sprintf(getBaseRequest()+Inventory, accountId))
}

func requestCompartments(accountId uint32) requests.Request[[]CompartmentRestModel] {
	return rest.MakeGetRequest[[]CompartmentRestModel](fmt.Sprintf(getBaseRequest()+Compartments, accountId))
}
