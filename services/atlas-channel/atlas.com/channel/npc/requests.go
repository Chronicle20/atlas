package npc

import (
	"atlas-channel/rest"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/Chronicle20/atlas-tenant"
	"os"
)

const (
	npcsInMap           = "maps/%d/npcs"
	npcsInMapByObjectId = npcsInMap + "?objectId=%d"
)

func getBaseRequest() string {
	return os.Getenv("GAME_DATA_SERVICE_URL")
}

func requestNPCsInMap(ctx context.Context, tenant tenant.Model) func(mapId uint32) requests.Request[[]RestModel] {
	return func(mapId uint32) requests.Request[[]RestModel] {
		return rest.MakeGetRequest[[]RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+npcsInMap, mapId))
	}
}

func requestNPCsInMapByObjectId(ctx context.Context, tenant tenant.Model) func(mapId uint32, objectId uint32) requests.Request[[]RestModel] {
	return func(mapId uint32, objectId uint32) requests.Request[[]RestModel] {
		return rest.MakeGetRequest[[]RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+npcsInMapByObjectId, mapId, objectId))
	}
}
