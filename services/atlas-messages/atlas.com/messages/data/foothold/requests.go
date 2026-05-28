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

func getInMap(mapId _map.Id, x int16, y int16) requests.Request[RestModel] {
	i := PositionRestModel{X: x, Y: y}
	return requests.PostRequest[RestModel](fmt.Sprintf(getBaseRequest()+footholdBelowResource, mapId), i)
}
