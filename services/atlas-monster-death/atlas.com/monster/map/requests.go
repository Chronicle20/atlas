package _map

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	mapResource                   = "worlds/%d/channels/%d/maps/%d"
	mapInstanceResource           = mapResource + "/instances/%s"
	mapInstanceCharactersResource = mapInstanceResource + "/characters/"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MAPS")
}

// charactersInFieldUrl returns the list URL for the characters currently in
// one map instance. It is a bare URL (not a requests.Request) because the
// list is now paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func charactersInFieldUrl(ctx context.Context, f field.Model) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+mapInstanceCharactersResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String()), nil
}
