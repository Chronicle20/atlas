package channel

import (
	"atlas-world/kafka"
	"atlas-world/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func emitChannelServerStarted(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, ipAddress string, port int) {
	return func(worldId byte, channelId byte, ipAddress string, port int) {
		emitChannelServerEvent(l, span, tenant)(worldId, channelId, EventChannelStatusType, ipAddress, port)
	}
}

func emitChannelServerShutdown(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, ipAddress string, port int) {
	return func(worldId byte, channelId byte, ipAddress string, port int) {
		emitChannelServerEvent(l, span, tenant)(worldId, channelId, EventChannelStatusTypeShutdown, ipAddress, port)
	}
}

func emitChannelServerEvent(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(worldId byte, channelId byte, status string, ipAddress string, port int) {
	p := producer.ProduceEvent(l, span, kafka.LookupTopic(l)(EnvEventTopicChannelStatus))
	return func(worldId byte, channelId byte, status string, ipAddress string, port int) {
		event := &channelStatusEvent{
			Tenant:    tenant,
			WorldId:   worldId,
			ChannelId: channelId,
			Type:      status,
			IpAddress: ipAddress,
			Port:      port,
		}
		p([]byte(tenant.Id.String()), event)
	}
}

func emitChannelServerStatusCommand(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) {
	p := producer.ProduceEvent(l, span, kafka.LookupTopic(l)(EnvCommandTopicChannelStatus))
	c := &channelStatusCommand{
		Tenant: tenant,
		Type:   CommandChannelStatusType,
	}
	p([]byte(tenant.Id.String()), c)
}
