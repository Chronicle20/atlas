package mts

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

const (
	// holdingsResource lists a character's take-home holdings, optionally scoped
	// to a world. atlas-mts exposes GET /characters/{characterId}/mts/holding.
	holdingsResource = "characters/%d/mts/holding?worldId=%d"
)

func getBaseRequest() string {
	return requests.RootUrl("MTS")
}

// RequestHoldings retrieves a character's holdings (optionally world-scoped)
// from atlas-mts so expansion can capture the item snapshot for the
// accept_to_character step. Mirrors cashshop.RequestCompartment.
func RequestHoldings(l logrus.FieldLogger, ctx context.Context) func(characterId uint32, worldId byte) ([]HoldingRestModel, error) {
	return func(characterId uint32, worldId byte) ([]HoldingRestModel, error) {
		url := fmt.Sprintf(getBaseRequest()+holdingsResource, characterId, worldId)
		return requests.GetRequest[[]HoldingRestModel](url)(l, ctx)
	}
}
