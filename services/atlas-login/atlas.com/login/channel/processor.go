package channel

import (
	"atlas-login/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func ByWorldModelProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte) model.SliceProvider[Model] {
	return func(worldId byte) model.SliceProvider[Model] {
		return requests.SliceProvider[RestModel, Model](l)(requestChannelsForWorld(l, span, tenant)(worldId), Extract)
	}
}
