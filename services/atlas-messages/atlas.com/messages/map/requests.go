package _map

import (
	"atlas-messages/rest"
	"atlas-messages/tenant"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	getMap = "maps/%d"
)

func getBaseRequest() string {
	return os.Getenv("GAME_DATA_SERVICE_URL")
}

func requestMap(ctx context.Context, tenant tenant.Model) func(mapId uint32) requests.Request[RestModel] {
	return func(mapId uint32) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+getMap, mapId))
	}
}
