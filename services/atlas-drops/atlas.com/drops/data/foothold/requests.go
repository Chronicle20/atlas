package foothold

import (
	"fmt"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	footholdBelowResource = "data/maps/%d/footholds/below"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

// requestBelow POSTs the search point to atlas-data and yields the foothold
// segment directly below it (404/500 when none exists).
func requestBelow(mapId _map.Id, x int16, y int16) requests.Request[FootholdRestModel] {
	i := PositionRestModel{X: x, Y: y}
	return requests.PostRequest[FootholdRestModel](fmt.Sprintf(getBaseRequest()+footholdBelowResource, mapId), i)
}
