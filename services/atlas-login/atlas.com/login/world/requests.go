package world

import (
	"atlas-login/rest"
	"atlas-login/tenant"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"os"
)

const (
	WorldsResource = "worlds/"
	WorldsById     = WorldsResource + "%d"
)

func getBaseRequest() string {
	return os.Getenv("WORLD_SERVICE_URL")
}

func requestWorlds(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](l, span, tenant)(getBaseRequest() + WorldsResource)
}

func requestWorld(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte) requests.Request[RestModel] {
	return func(worldId byte) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+WorldsById, worldId))
	}
}
