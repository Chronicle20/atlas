package cashshop

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	compartmentsResource    = "accounts/%d/cash-shop/inventory/compartments?type=%d"
	compartmentByIdResource = "accounts/%d/cash-shop/inventory/compartments/%s"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "CASHSHOP")
}

// RequestCompartment retrieves a compartment with its assets from the cash shop service
func RequestCompartment(l logrus.FieldLogger, ctx context.Context) func(accountId uint32, compartmentType byte) (CompartmentRestModel, error) {
	return func(accountId uint32, compartmentType byte) (CompartmentRestModel, error) {
		root, err := getBaseRequest(ctx)
		if err != nil {
			return CompartmentRestModel{}, err
		}
		url := fmt.Sprintf(root+compartmentsResource, accountId, compartmentType)
		return requests.GetRequest[CompartmentRestModel](url)(l, ctx)
	}
}
