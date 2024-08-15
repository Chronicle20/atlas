package character

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
	Resource = "characters"
	ById     = Resource + "/%d"
)

func getBaseRequest() string {
	return os.Getenv("CHARACTER_SERVICE_URL")
}

func requestById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(id uint32) requests.Request[RestModel] {
	return func(id uint32) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+ById, id))
	}
}
