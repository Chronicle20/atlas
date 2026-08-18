package foothold

import (
	"context"
	"fmt"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	footholdBelowResource = "data/maps/%d/footholds/below"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DATA")
}

func getInMap(ctx context.Context, mapId _map.Id, x int16, y int16) requests.Request[RestModel] {
	i := PositionRestModel{X: x, Y: y}
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.PostRequest[RestModel](fmt.Sprintf(root+footholdBelowResource, mapId), i)
}
