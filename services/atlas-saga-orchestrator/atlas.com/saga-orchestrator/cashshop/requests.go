package cashshop

import (
	"atlas-saga-orchestrator/rest"
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

const (
	compartmentsResource    = "accounts/%d/cash-shop/inventory/compartments?type=%d"
	compartmentByIdResource = "accounts/%d/cash-shop/inventory/compartments/%s"
)

func getBaseRequest() string {
	return requests.RootUrl("CASHSHOP")
}

// RequestCompartment retrieves a compartment with its assets from the cash shop service
func RequestCompartment(l logrus.FieldLogger, ctx context.Context) func(accountId uint32, compartmentType byte) (CompartmentRestModel, error) {
	return func(accountId uint32, compartmentType byte) (CompartmentRestModel, error) {
		url := fmt.Sprintf(getBaseRequest()+compartmentsResource, accountId, compartmentType)
		return rest.MakeGetRequest[CompartmentRestModel](url)(l, ctx)
	}
}
