package inventory

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource          = "characters/%d/inventory"
	CompartmentAssets = "characters/%d/inventory/compartments/%s/assets"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "INVENTORY")
}

func requestInventory(ctx context.Context, characterId uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+Resource, characterId))
}

// compartmentAssetsUrl returns the list URL for a compartment's assets. It is
// a bare URL (not a requests.Request) because the list is now paginated
// server-side (task-117) and consumed via requests.DrainProvider, which
// appends its own page[number]/page[size] query params per request.
func compartmentAssetsUrl(ctx context.Context, characterId uint32, compartmentId string) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+CompartmentAssets, characterId, compartmentId), nil
}
