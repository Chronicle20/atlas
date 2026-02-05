package channel

import (
	channel2 "atlas-world/kafka/message/channel"
	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
)

func StartedEventProvider(tenant tenant.Model, worldId world.Id, channelId channel.Id, ipAddress string, port int, currentCapacity uint32, maxCapacity uint32) model.Provider[[]kafka.Message] {
	return EventProvider(tenant, worldId, channelId, channel.StatusTypeStarted, ipAddress, port, currentCapacity, maxCapacity)
}

func ShutdownEventProvider(tenant tenant.Model, worldId world.Id, channelId channel.Id, ipAddress string, port int, currentCapacity uint32, maxCapacity uint32) model.Provider[[]kafka.Message] {
	return EventProvider(tenant, worldId, channelId, channel.StatusTypeShutdown, ipAddress, port, currentCapacity, maxCapacity)
}

func EventProvider(tenant tenant.Model, worldId world.Id, channelId channel.Id, status channel.StatusType, ipAddress string, port int, currentCapacity uint32, maxCapacity uint32) model.Provider[[]kafka.Message] {
	key := []byte(tenant.Id().String())
	value := &channel2.StatusEvent{
		Type:            status,
		WorldId:         worldId,
		ChannelId:       channelId,
		IpAddress:       ipAddress,
		Port:            port,
		CurrentCapacity: currentCapacity,
		MaxCapacity:     maxCapacity,
	}
	return producer.SingleMessageProvider(key, value)
}

func StatusCommandProvider(tenant tenant.Model) model.Provider[[]kafka.Message] {
	key := []byte(tenant.Id().String())
	value := &channel2.StatusCommand{
		Type: channel2.CommandTypeStatusRequest,
	}
	return producer.SingleMessageProvider(key, value)
}
