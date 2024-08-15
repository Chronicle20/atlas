package _map

import (
	"atlas-messages/rest"
	"atlas-messages/tenant"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"os"
)

const (
	getMap = "maps/%d"
)

func getBaseRequest() string {
	return os.Getenv("GAME_DATA_SERVICE_URL")
}

func requestMap(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(mapId uint32) requests.Request[RestModel] {
	return func(mapId uint32) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+getMap, mapId))
	}
}
