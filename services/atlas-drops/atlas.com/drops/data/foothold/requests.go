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

// requestBelow POSTs the search point to atlas-data and yields the foothold
// segment directly below it (404/500 when none exists).
func requestBelow(ctx context.Context, mapId _map.Id, x int16, y int16) requests.Request[FootholdRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[FootholdRestModel](err)
	}
	i := PositionRestModel{X: x, Y: y}
	return requests.PostRequest[FootholdRestModel](fmt.Sprintf(root+footholdBelowResource, mapId), i)
}
