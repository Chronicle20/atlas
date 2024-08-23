package statistics

import (
	"atlas-character/rest"
	"atlas-character/tenant"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	itemInformationResource = "equipment/"
	itemInformationById     = itemInformationResource + "%d"
)

func getBaseRequest() string {
	return os.Getenv("GAME_DATA_SERVICE_URL")
}

func requestById(ctx context.Context, tenant tenant.Model) func(id uint32) requests.Request[RestModel] {
	return func(id uint32) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+itemInformationById, id))
	}
}
