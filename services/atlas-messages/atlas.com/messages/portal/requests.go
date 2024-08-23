package portal

import (
	"atlas-messages/rest"
	"atlas-messages/tenant"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	portalsInMap = "maps/%d/portals"
)

func getBaseRequest() string {
	return os.Getenv("GAME_DATA_SERVICE_URL")
}

func requestAll(ctx context.Context, tenant tenant.Model) func(mapId uint32) requests.Request[[]RestModel] {
	return func(mapId uint32) requests.Request[[]RestModel] {
		return rest.MakeGetRequest[[]RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+portalsInMap, mapId))
	}
}
