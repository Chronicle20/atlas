package position

import (
	"context"
	"fmt"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	positionsResource = "data/maps/%d/footholds/below"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DATA")
}

func getInMap(ctx context.Context, mapId _map.Id, x int16, y int16) requests.Request[FootholdRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[FootholdRestModel](err)
	}
	i := PositionRestModel{
		X: x,
		Y: y,
	}
	return requests.PostRequest[FootholdRestModel](fmt.Sprintf(root+positionsResource, mapId), i)
}
