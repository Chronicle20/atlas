package data

import (
	"atlas-character-factory/rest"
	"atlas-character-factory/tenant"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"os"
)

const (
	itemInformationResource        = "equipment/"
	itemInformationById            = itemInformationResource + "%d"
	itemDestinationSlotInformation = itemInformationById + "/slots"
)

func getBaseRequest() string {
	return os.Getenv("GAME_DATA_SERVICE_URL")
}

func requestById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(id uint32) requests.Request[[]RestModel] {
	return func(id uint32) requests.Request[[]RestModel] {
		return rest.MakeGetRequest[[]RestModel](l, span, tenant)(fmt.Sprintf(getBaseRequest()+itemDestinationSlotInformation, id))
	}
}
