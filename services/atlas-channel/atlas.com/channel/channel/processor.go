package channel

import (
	"atlas-channel/tenant"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"strconv"
)

func Register(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, ipAddress string, port string) error {
	return func(worldId byte, channelId byte, ipAddress string, portStr string) error {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			l.WithError(err).Errorf("Port [%s] is not a valid number.", portStr)
			return err
		}
		return registerChannel(l, span, tenant)(worldId, channelId, ipAddress, port)
	}
}

func Unregister(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte) error {
	return func(worldId byte, channelId byte) error {
		return unregisterChannel(l, span, tenant)(worldId, channelId)
	}
}

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
