package inventory

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// compartmentByType fetches one inventory compartment (by type) with its assets
// included, mirroring atlas-summons' inventory client.
const compartmentByType = "characters/%d/inventory/compartments?type=%d&include=assets"

var baseURLProvider = func(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "INVENTORY")
}

func getBaseRequest(ctx context.Context) (string, error) {
	return baseURLProvider(ctx)
}

func requestCompartmentByType(ctx context.Context, characterId uint32, inventoryType inventory.Type) requests.Request[CompartmentRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[CompartmentRestModel](err)
	}
	return requests.GetRequest[CompartmentRestModel](fmt.Sprintf(root+compartmentByType, characterId, inventoryType))
}

// SetBaseURLForTest swaps the base URL for httptest-backed tests. Only call
// from a test; production uses the env-driven RootUrlFor("INVENTORY")
// default. The injected closure ignores ctx -- tests always exercise the
// fixed httptest URL regardless of any environment on the context.
func SetBaseURLForTest(url string) func() {
	prev := baseURLProvider
	baseURLProvider = func(_ context.Context) (string, error) { return url + "/api/", nil }
	return func() { baseURLProvider = prev }
}
