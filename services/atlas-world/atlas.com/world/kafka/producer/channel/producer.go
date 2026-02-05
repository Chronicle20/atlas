package channel

import (
	channel2 "atlas-world/kafka/message/channel"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/segmentio/kafka-go"
)

func StartedEventProvider(tenant tenant.Model, ch channel.Model, ipAddress string, port int, currentCapacity uint32, maxCapacity uint32) model.Provider[[]kafka.Message] {
	return EventProvider(tenant, ch, channel.StatusTypeStarted, ipAddress, port, currentCapacity, maxCapacity)
}

func ShutdownEventProvider(tenant tenant.Model, ch channel.Model, ipAddress string, port int, currentCapacity uint32, maxCapacity uint32) model.Provider[[]kafka.Message] {
	return EventProvider(tenant, ch, channel.StatusTypeShutdown, ipAddress, port, currentCapacity, maxCapacity)
}

func EventProvider(tenant tenant.Model, ch channel.Model, status channel.StatusType, ipAddress string, port int, currentCapacity uint32, maxCapacity uint32) model.Provider[[]kafka.Message] {
	key := []byte(tenant.Id().String())
	value := &channel2.StatusEvent{
		Type:            status,
		WorldId:         ch.WorldId(),
		ChannelId:       ch.Id(),
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
