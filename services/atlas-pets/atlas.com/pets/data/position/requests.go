package position

import (
	"atlas-pets/rest"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	positionsResource = "data/maps/%d/footholds/below"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func getInMap(mapId uint32, x int16, y int16) requests.Request[FootholdRestModel] {
	i := PositionRestModel{
		X: x,
		Y: y,
	}
	return rest.MakePostRequest[FootholdRestModel](fmt.Sprintf(getBaseRequest()+positionsResource, mapId), i)
}
