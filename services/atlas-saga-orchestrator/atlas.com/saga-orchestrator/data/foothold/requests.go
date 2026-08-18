package foothold

import (
	"context"
	"fmt"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	footholdBelowPath = "data/maps/%d/footholds/below"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DATA")
}

func requestFootholdBelow(ctx context.Context, mapId _map.Id, input PositionInputRestModel) requests.Request[FootholdRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[FootholdRestModel](err)
	}
	return requests.PostRequest[FootholdRestModel](fmt.Sprintf(root+footholdBelowPath, mapId), input)
}
