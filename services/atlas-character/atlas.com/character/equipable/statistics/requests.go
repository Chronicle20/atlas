package statistics

import (
	"atlas-character/rest"
	"atlas-character/tenant"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"os"
)

const (
	equipmentResource = "equipment"
	equipResource     = equipmentResource + "/%d"
)

func getBaseRequest() string {
	return os.Getenv("EQUIPABLE_SERVICE_URL")
}

func requestCreate(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(itemId uint32) requests.Request[RestModel] {
	return func(itemId uint32) requests.Request[RestModel] {
		input := &RestModel{
			ItemId: itemId,
		}
		return rest.MakePostRequest[RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+equipmentResource), input)
	}
}

func requestById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(equipmentId uint32) requests.Request[RestModel] {
	return func(equipmentId uint32) requests.Request[RestModel] {
		return rest.MakeGetRequest[RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+equipResource, equipmentId))
	}
}

func deleteById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(equipmentId uint32) requests.EmptyBodyRequest {
	return func(equipmentId uint32) requests.EmptyBodyRequest {
		return rest.MakeDeleteRequest(l, span, tenant)(fmt.Sprintf(getBaseRequest()+equipResource, equipmentId))
	}
}
