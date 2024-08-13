package account

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
	AccountsResource = "accounts"
	AccountsByName   = AccountsResource + "?name=%s"
	AccountsById     = AccountsResource + "/%d"
	Update           = AccountsResource + "/%d"
)

func getBaseRequest() string {
	return os.Getenv("ACCOUNT_SERVICE_URL")
}

func requestAccounts(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) requests.Request[[]RestModel] {
	return rest.MakeGetRequest[[]RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest() + AccountsResource))
}

func requestAccountByName(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(name string) requests.Request[RestModel] {
	return func(name string) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+AccountsByName, name))
	}
}

func requestAccountById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(id uint32) requests.Request[RestModel] {
	return func(id uint32) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+AccountsById, id))
	}
}

func requestUpdate(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(m Model) requests.Request[RestModel] {
	return func(m Model) requests.Request[RestModel] {
		im, _ := Transform(m)
		return rest.MakePatchRequest[RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+Update, m.id), im)
	}
}
