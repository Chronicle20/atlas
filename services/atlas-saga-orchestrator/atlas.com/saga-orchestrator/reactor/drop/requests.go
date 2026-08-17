package drop

import (
	"context"
	"fmt"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	reactorDropsPath    = "reactors/%d/drops"
	mapDropPositionPath = "data/maps/%d/drops/position"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DROP_INFORMATION")
}

func getDataBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DATA")
}

func requestReactorDrops(ctx context.Context, reactorId uint32) requests.Request[ReactorRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[ReactorRestModel](err)
	}
	return requests.GetRequest[ReactorRestModel](fmt.Sprintf(root+reactorDropsPath, reactorId))
}

func requestDropPosition(ctx context.Context, mapId _map.Id, initialX, initialY, fallbackX, fallbackY int16) requests.Request[PositionRestModel] {
	input := DropPositionInputModel{
		InitialX:  initialX,
		InitialY:  initialY,
		FallbackX: fallbackX,
		FallbackY: fallbackY,
	}
	root, err := getDataBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[PositionRestModel](err)
	}
	return requests.PostRequest[PositionRestModel](fmt.Sprintf(root+mapDropPositionPath, mapId), input)
}
