package position

import (
	"atlas-consumables/rest"
	"fmt"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	positionsResource = "data/maps/%d/drops/position"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func getInMap(mapId _map.Id, initialX int16, initialY int16, fallbackX int16, fallbackY int16) requests.Request[RestModel] {
	i := DropPositionRestModel{
		InitialX:  initialX,
		InitialY:  initialY,
		FallbackX: fallbackX,
		FallbackY: fallbackY,
	}
	return rest.MakePostRequest[RestModel](fmt.Sprintf(getBaseRequest()+positionsResource, mapId), i)
}
