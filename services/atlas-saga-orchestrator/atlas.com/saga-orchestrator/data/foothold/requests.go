package foothold

import (
	"atlas-saga-orchestrator/rest"
	"fmt"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-rest/requests"
)

const (
	footholdBelowPath = "data/maps/%d/footholds/below"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

func requestFootholdBelow(mapId _map.Id, input PositionInputRestModel) requests.Request[FootholdRestModel] {
	return rest.MakePostRequest[FootholdRestModel](fmt.Sprintf(getBaseRequest()+footholdBelowPath, mapId), input)
}
