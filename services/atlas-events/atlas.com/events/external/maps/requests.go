package maps

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	mapResource           = "worlds/%d/channels/%d/maps/%d/instances/%s"
	mapCharactersResource = mapResource + "/characters/"
)

// getBaseRequest resolves the ingress for the environment carried on ctx
// rather than the process-wide baseline (task-232 FR-3.5/G4) — see the
// transports sibling for why the context-free requests.RootUrl would send
// this call into main from an ephemeral-environment pod.
func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "MAPS")
}

// charactersInMapUrl returns the list URL for the characters currently in
// one map instance. It is a bare URL (not a requests.Request) because the
// list is paginated server-side (task-117) and consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func charactersInMapUrl(ctx context.Context, f field.Model) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+mapCharactersResource, f.WorldId(), f.ChannelId(), f.MapId(), f.Instance()), nil
}
