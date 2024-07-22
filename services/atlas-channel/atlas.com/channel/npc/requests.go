package npc

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
	npcsInMap           = "maps/%d/npcs"
	npcsInMapByObjectId = npcsInMap + "?objectId=%d"
)

func getBaseRequest() string {
	return os.Getenv("GAME_DATA_SERVICE_URL")
}

func requestNPCsInMap(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(mapId uint32) requests.Request[[]RestModel] {
	return func(mapId uint32) requests.Request[[]RestModel] {
		return rest.MakeGetRequest[[]RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+npcsInMap, mapId))
	}
}

func requestNPCsInMapByObjectId(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(mapId uint32, objectId uint32) requests.Request[[]RestModel] {
	return func(mapId uint32, objectId uint32) requests.Request[[]RestModel] {
		return rest.MakeGetRequest[[]RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+npcsInMapByObjectId, mapId, objectId))
	}
}
