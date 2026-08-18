package reactor

import (
	"context"
	"fmt"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	mapsResource     = "data/maps/"
	reactorsResource = mapsResource + "%d/reactors"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DATA")
}

// reactorsUrl is a bare URL (not a requests.Request) because the list is
// now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func reactorsUrl(ctx context.Context, mapId _map.Id) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+reactorsResource, mapId), nil
}
