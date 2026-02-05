package drop

import (
	"atlas-saga-orchestrator/rest"
	"fmt"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	reactorDropsPath    = "reactors/%d/drops"
	mapDropPositionPath = "data/maps/%d/drops/position"
)

func getBaseRequest() string {
	return requests.RootUrl("DROP_INFORMATION")
}

func getDataBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestReactorDrops(reactorId uint32) requests.Request[ReactorRestModel] {
	return rest.MakeGetRequest[ReactorRestModel](fmt.Sprintf(getBaseRequest()+reactorDropsPath, reactorId))
}

func requestDropPosition(mapId _map.Id, initialX, initialY, fallbackX, fallbackY int16) requests.Request[PositionRestModel] {
	input := DropPositionInputModel{
		InitialX:  initialX,
		InitialY:  initialY,
		FallbackX: fallbackX,
		FallbackY: fallbackY,
	}
	return rest.MakePostRequest[PositionRestModel](fmt.Sprintf(getDataBaseRequest()+mapDropPositionPath, mapId), input)
}
