package character

import (
	"atlas-messages/rest"
	"atlas-messages/tenant"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"os"
)

const (
	Resource = "characters"
	ById     = Resource + "/%d"
)

func getBaseRequest() string {
	return os.Getenv("CHARACTER_SERVICE_URL")
}

func requestById(ctx context.Context, tenant tenant.Model) func(id uint32) requests.Request[RestModel] {
	return func(id uint32) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+ById, id))
	}
}
