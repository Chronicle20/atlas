package object

import (
	"context"
	"fmt"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	mapsResource    = "data/maps/"
	objectsResource = mapsResource + "%d/objects"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DATA")
}

// objectsUrl is a bare URL (not a requests.Request) because the collection is
// consumed via requests.DrainProvider, which appends its own
// page[number]/page[size] query params per request.
func objectsUrl(ctx context.Context, mapId _map.Id) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+objectsResource, mapId), nil
}
