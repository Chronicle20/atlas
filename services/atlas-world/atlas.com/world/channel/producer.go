package channel

import (
	"atlas-world/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func emitChannelServerStarted(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, ipAddress string, port int) {
	return func(worldId byte, channelId byte, ipAddress string, port int) {
		emitChannelServerEvent(l, span, tenant)(worldId, channelId, EventStatusStarted, ipAddress, port)
	}
}

func emitChannelServerShutdown(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, ipAddress string, port int) {
	return func(worldId byte, channelId byte, ipAddress string, port int) {
		emitChannelServerEvent(l, span, tenant)(worldId, channelId, EventStatusShutdown, ipAddress, port)
	}
}

func emitChannelServerEvent(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, status string, ipAddress string, port int) {
	p := producer.ProduceEvent(l, span, lookupTopic(l)(topicTokenStatus))
	return func(worldId byte, channelId byte, status string, ipAddress string, port int) {
		event := &channelServerEvent{
			Tenant:    tenant,
			WorldId:   worldId,
			ChannelId: channelId,
			Status:    status,
			IpAddress: ipAddress,
			Port:      port,
		}
		p([]byte(tenant.Id().String()), event)
	}
}
