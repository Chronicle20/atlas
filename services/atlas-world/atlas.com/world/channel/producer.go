package channel

import (
	"atlas-world/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func emitChannelServerStarted(tenant tenant.Model, worldId byte, channelId byte, ipAddress string, port int) model.Provider[[]kafka.Message] {
	return emitChannelServerEvent(tenant, worldId, channelId, EventChannelStatusType, ipAddress, port)
}

func emitChannelServerShutdown(tenant tenant.Model, worldId byte, channelId byte, ipAddress string, port int) model.Provider[[]kafka.Message] {
	return emitChannelServerEvent(tenant, worldId, channelId, EventChannelStatusTypeShutdown, ipAddress, port)
}

func emitChannelServerEvent(tenant tenant.Model, worldId byte, channelId byte, status string, ipAddress string, port int) model.Provider[[]kafka.Message] {
	key := []byte(tenant.Id.String())
	value := &channelStatusEvent{
		Tenant:    tenant,
		WorldId:   worldId,
		ChannelId: channelId,
		Type:      status,
		IpAddress: ipAddress,
		Port:      port,
	}
	return producer.SingleMessageProvider(key, value)
}

func emitChannelServerStatusCommand(tenant tenant.Model) model.Provider[[]kafka.Message] {
	key := []byte(tenant.Id.String())
	value := &channelStatusCommand{
		Tenant: tenant,
		Type:   CommandChannelStatusType,
	}
	return producer.SingleMessageProvider(key, value)
}
