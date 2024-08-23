package character

import (
	"atlas-login/rest"
	"atlas-login/tenant"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
	"os"
)

const (
	Resource          = "characters"
	ByAccountAndWorld = Resource + "?accountId=%d&worldId=%d&include=inventory"
	ByName            = Resource + "?name=%s&include=inventory"
	ById              = Resource + "/%d?include=inventory"
)

func getBaseRequest() string {
	return os.Getenv("CHARACTER_SERVICE_URL")
}

func requestByAccountAndWorld(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(accountId uint32, worldId byte) requests.Request[[]RestModel] {
	return func(accountId uint32, worldId byte) requests.Request[[]RestModel] {
		return rest.MakeGetRequest[[]RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+ByAccountAndWorld, accountId, worldId))
	}
}

func requestByName(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(name string) requests.Request[[]RestModel] {
	return func(name string) requests.Request[[]RestModel] {
		return rest.MakeGetRequest[[]RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+ByName, name))
	}
}

func requestById(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(id uint32) requests.Request[RestModel] {
	return func(id uint32) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](ctx, tenant)(fmt.Sprintf(getBaseRequest()+ById, id))
	}
}

func requestDelete(l logrus.FieldLogger, ctx context.Context, tenant tenant.Model) func(id uint32) requests.EmptyBodyRequest {
	return func(id uint32) requests.EmptyBodyRequest {
		return rest.MakeDeleteRequest(ctx, tenant)(fmt.Sprintf(getBaseRequest()+ById, id))
	}
}
