package information

import (
	"atlas-monsters/rest"
	"atlas-monsters/tenant"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"os"
)

const (
	monstersResource = "monsters"
	monsterResource  = monstersResource + "/%d"
)

func getBaseRequest() string {
	return os.Getenv("GAME_DATA_SERVICE_URL")
}

func requestById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(monsterId uint32) requests.Request[RestModel] {
	return func(monsterId uint32) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+monsterResource, monsterId))
	}
}
