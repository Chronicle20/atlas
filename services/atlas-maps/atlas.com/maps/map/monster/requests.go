package monster

import (
	"atlas-maps/rest"
	"atlas-maps/tenant"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"os"
)

const (
	mapsResource     = "maps/"
	monstersResource = mapsResource + "%d/monsters"
)

func getBaseRequest() string {
	return os.Getenv("GAME_DATA_SERVICE_URL")
}

func requestSpawnPoints(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(mapId uint32) requests.Request[[]RestModel] {
	return func(mapId uint32) requests.Request[[]RestModel] {
		return rest.MakeGetRequest[[]RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+monstersResource, mapId))
	}
}
