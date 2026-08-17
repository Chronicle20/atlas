package position

import (
	"context"
	"fmt"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	positionsResource = "data/maps/%d/drops/position"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DATA")
}

func getInMap(ctx context.Context, mapId _map.Id, initialX int16, initialY int16, fallbackX int16, fallbackY int16) requests.Request[RestModel] {
	i := DropPositionRestModel{
		InitialX:  initialX,
		InitialY:  initialY,
		FallbackX: fallbackX,
		FallbackY: fallbackY,
	}
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.PostRequest[RestModel](fmt.Sprintf(root+positionsResource, mapId), i)
}
