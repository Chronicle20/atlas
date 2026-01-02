package transport

import (
	"atlas-query-aggregator/rest"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-rest/requests"
	"fmt"
)

// getBaseRequest returns the base URL for transport requests
func getBaseRequest() string {
	return requests.RootUrl("TRANSPORTS") + "/transports/routes"
}

// requestRoutesByStartMap requests routes filtered by start map ID
// Uses JSON:API filter syntax: ?filter[startMapId]={mapId}
func requestRoutesByStartMap(mapId _map.Id) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](
		fmt.Sprintf(getBaseRequest()+"?filter[startMapId]=%d", mapId),
	)
}
