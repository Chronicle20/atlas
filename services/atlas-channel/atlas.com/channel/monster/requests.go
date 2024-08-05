package monster

import (
	"atlas-channel/rest"
	"atlas-channel/tenant"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"os"
)

const (
	mapMonstersResource = "worlds/%d/channels/%d/maps/%d/monsters"
)

func getBaseRequest() string {
	return os.Getenv("MONSTER_SERVICE_URL")
}

func requestInMap(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, mapId uint32) requests.Request[[]RestModel] {
	return func(worldId byte, channelId byte, mapId uint32) requests.Request[[]RestModel] {
		return rest.MakeGetRequest[[]RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+mapMonstersResource, worldId, channelId, mapId))
	}
}
