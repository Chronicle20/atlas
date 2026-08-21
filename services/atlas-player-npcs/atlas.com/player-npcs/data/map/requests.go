package _map

import (
	"context"
	"fmt"

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

func requestById(ctx context.Context, id uint32) requests.Request[RestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[RestModel](err)
	}
	return requests.GetRequest[RestModel](fmt.Sprintf(root+ById, id))
}

func requestGround(ctx context.Context, mapId uint32, points []GroundPointRestModel) requests.Request[[]GroundResultRestModel] {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return requests.ErrorRequest[[]GroundResultRestModel](err)
	}
	body := GroundRequestRestModel{Points: points}
	return requests.PostRequest[[]GroundResultRestModel](fmt.Sprintf(root+Ground, mapId), body)
}
