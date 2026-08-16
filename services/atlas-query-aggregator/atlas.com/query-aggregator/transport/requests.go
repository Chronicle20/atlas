package transport

import (
	"context"
	"fmt"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// getBaseRequest returns the base URL for transport requests
func getBaseRequest(ctx context.Context) (string, error) {
	root, err := requests.RootUrlFor(ctx, "TRANSPORTS")
	if err != nil {
		return "", err
	}
	return root + "/transports/routes", nil
}

// requestRoutesByStartMap requests routes filtered by start map ID
// Uses JSON:API filter syntax: ?filter[startMapId]={mapId}
func requestRoutesByStartMap(ctx context.Context, mapId _map.Id) requests.Request[[]RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]RestModel](err)
	}
	return requests.GetRequest[[]RestModel](
		fmt.Sprintf(root+"?filter[startMapId]=%d", mapId),
	)
}
