package information

import (
	"atlas-monsters/rest"
	"atlas-monsters/tenant"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	monstersResource = "monsters"
	monsterResource  = monstersResource + "/%d"
)

func getBaseRequest() string {
	return os.Getenv("GAME_DATA_SERVICE_URL")
}

func requestById(ctx context.Context, tenant tenant.Model) func(monsterId uint32) requests.Request[RestModel] {
	return func(monsterId uint32) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+monsterResource, monsterId))
	}
}
