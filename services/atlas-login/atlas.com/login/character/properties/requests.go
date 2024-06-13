package properties

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
	CharactersResource          = "characters/"
	CharactersByName            = CharactersResource + "?name=%s"
	CharactersForAccountByWorld = CharactersResource + "?accountId=%d&worldId=%d"
	CharactersById              = CharactersResource + "%d"
)

func getBaseRequest() string {
	return os.Getenv("ACCOUNT_SERVICE_URL")
}

func requestPropertiesByName(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(name string) requests.Request[RestModel] {
	return func(name string) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+CharactersByName, name))
	}
}

func requestPropertiesByAccountAndWorld(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(accountId uint32, worldId byte) requests.Request[RestModel] {
	return func(accountId uint32, worldId byte) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+CharactersForAccountByWorld, accountId, worldId))
	}
}

func requestPropertiesById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(characterId uint32) requests.Request[RestModel] {
	return func(characterId uint32) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+CharactersById, characterId))
	}
}
