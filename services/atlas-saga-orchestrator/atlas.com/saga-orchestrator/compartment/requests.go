package compartment

import (
	"atlas-saga-orchestrator/rest"
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

const (
	compartmentsResource = "characters/%d/inventory/compartments?type=%d"
)

func getBaseRequest() string {
	return requests.RootUrl("INVENTORY")
}

// RequestCompartment retrieves a compartment with its assets from the character inventory service
func RequestCompartment(l logrus.FieldLogger, ctx context.Context) func(characterId uint32, inventoryType byte) (CompartmentRestModel, error) {
	return func(characterId uint32, inventoryType byte) (CompartmentRestModel, error) {
		url := fmt.Sprintf(getBaseRequest()+compartmentsResource, characterId, inventoryType)
		return rest.MakeGetRequest[CompartmentRestModel](url)(l, ctx)
	}
}
