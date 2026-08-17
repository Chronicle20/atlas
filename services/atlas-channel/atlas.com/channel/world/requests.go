package world

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	WorldsResource = "worlds/"
	WorldsById     = WorldsResource + "%d"
	// WorldsList is the collection resource, without WorldsResource's trailing
	// slash — atlas-world's paginated worlds endpoint.
	WorldsList = "worlds"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "WORLDS")
}

// worldsUrl is a bare URL (not a requests.Request) because the list is
// paginated server-side (task-117) and consumed via requests.DrainProvider,
// which appends its own page[number]/page[size] query params per request.
// Mirrors atlas-login's world/requests.go, minus its ?include=channels —
// this service's only list consumer (the cash-shop world-transfer name list)
// needs names, not channels, and Extract tolerates an empty channel set.
func worldsUrl(ctx context.Context) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return root + WorldsList, nil
}

func requestWorld(ctx context.Context, worldId world.Id) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+WorldsById, worldId))
}
