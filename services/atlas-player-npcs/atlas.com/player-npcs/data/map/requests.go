package map_

import (
	"context"
	"fmt"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "data/maps"
	ById     = Resource + "/%d"
	Ground   = ById + "/ground"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "DATA")
}

func requestById(ctx context.Context, id _map.Id) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+ById, id))
}

func requestGround(ctx context.Context, mapId _map.Id, points []GroundPointRestModel) requests.Request[[]GroundResultRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]GroundResultRestModel](err)
	}
	body := GroundRequestRestModel{Points: points}
	return requests.PostRequest[[]GroundResultRestModel](fmt.Sprintf(root+Ground, mapId), body)
}
