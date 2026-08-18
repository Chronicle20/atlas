package mts

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	// holdingsResource lists a character's take-home holdings, optionally scoped
	// to a world. atlas-mts exposes GET /characters/{characterId}/mts/holding.
	holdingsResource = "characters/%d/mts/holding?worldId=%d"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MTS")
}

// holdingsUrl returns the list URL for a character's holdings (optionally
// world-scoped). It is a bare URL (not a requests.Request) because the list is
// now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size] query
// params per request.
func holdingsUrl(ctx context.Context, characterId uint32, worldId byte) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+holdingsResource, characterId, worldId), nil
}

// identityHolding is the no-op transformer requests.DrainProvider requires:
// this consumer has no separate domain Model, so A == M == HoldingRestModel.
func identityHolding(m HoldingRestModel) (HoldingRestModel, error) { return m, nil }

// RequestHoldings retrieves a character's holdings (optionally world-scoped)
// from atlas-mts so expansion can capture the item snapshot for the
// accept_to_character step. The upstream list is now paginated (task-117);
// expandWithdrawFromMts linear-searches the result by HoldingId, so it needs
// the complete set — this drains every page rather than fetching just the
// first.
func RequestHoldings(l logrus.FieldLogger, ctx context.Context) func(characterId uint32, worldId byte) ([]HoldingRestModel, error) {
	return func(characterId uint32, worldId byte) ([]HoldingRestModel, error) {
		url, err := holdingsUrl(ctx, characterId, worldId)
		if err != nil {
			return nil, err
		}
		return requests.DrainProvider[HoldingRestModel, HoldingRestModel](l, ctx)(url, 250, identityHolding, model.Filters[HoldingRestModel]())()
	}
}
