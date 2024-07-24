package account

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
	AccountsResource = "accounts"
	AccountsById     = AccountsResource + "/%d"
)

func getBaseRequest() string {
	return os.Getenv("ACCOUNT_SERVICE_URL")
}

func requestAccountById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(id uint32) requests.Request[RestModel] {
	return func(id uint32) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+AccountsById, id))
	}
}
