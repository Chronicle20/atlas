package channel

import (
	"atlas-channel/tenant"
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
