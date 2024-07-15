package channel

import (
	"atlas-login/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func byIdModelProvider(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte) model.Provider[Model] {
	return func(worldId byte, channelId byte) model.Provider[Model] {
		return requests.Provider[RestModel, Model](l)(requestChannel(l, span, tenant)(worldId, channelId), Extract)
	}
}

func GetById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte) (Model, error) {
	return func(worldId byte, channelId byte) (Model, error) {
		return byIdModelProvider(l, span, tenant)(worldId, channelId)()
	}
}
