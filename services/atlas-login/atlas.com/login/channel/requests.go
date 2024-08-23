package channel

import (
	"atlas-login/rest"
	"atlas-login/tenant"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	WorldsResource = "worlds/"
	WorldsById     = WorldsResource + "%d"
	Resource       = WorldsById + "/channels"
)

func getBaseRequest() string {
	return os.Getenv("WORLD_SERVICE_URL")
}

func requestChannelsForWorld(ctx context.Context, tenant tenant.Model) func(worldId byte) requests.Request[[]RestModel] {
	return func(worldId byte) requests.Request[[]RestModel] {
		return rest.MakeGetRequest[[]RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+Resource, worldId))
	}
}
