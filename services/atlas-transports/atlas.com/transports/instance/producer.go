package instance

import (
	character2 "atlas-transports/kafka/message/character"
	it "atlas-transports/kafka/message/instance_transport"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func changeMapProvider(worldId world.Id, channelId channel.Id, characterId uint32, targetMapId _map.Id, instance uuid.UUID, portalId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.ChangeMapBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        character2.CommandCharacterChangeMap,
		Body: character2.ChangeMapBody{
			ChannelId: channelId,
			MapId:     targetMapId,
			Instance:  instance,
			PortalId:  portalId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func warpToTransitMapProvider(f field.Model, characterId uint32, transitMapId _map.Id, instanceId uuid.UUID) model.Provider[[]kafka.Message] {
	return changeMapProvider(f.WorldId(), f.ChannelId(), characterId, transitMapId, instanceId, 0)
}

func warpToDestinationProvider(worldId world.Id, channelId channel.Id, characterId uint32, destinationMapId _map.Id) model.Provider[[]kafka.Message] {
	return changeMapProvider(worldId, channelId, characterId, destinationMapId, uuid.Nil, 0)
}

func warpToStartMapProvider(worldId world.Id, channelId channel.Id, characterId uint32, startMapId _map.Id) model.Provider[[]kafka.Message] {
	return changeMapProvider(worldId, channelId, characterId, startMapId, uuid.Nil, 0)
}

func startedEventProvider(worldId world.Id, characterId uint32, routeId uuid.UUID, instanceId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &it.Event[it.StartedEventBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        it.EventTypeStarted,
		Body: it.StartedEventBody{
			RouteId:    routeId,
			InstanceId: instanceId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func completedEventProvider(worldId world.Id, characterId uint32, routeId uuid.UUID, instanceId uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &it.Event[it.CompletedEventBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        it.EventTypeCompleted,
		Body: it.CompletedEventBody{
			RouteId:    routeId,
			InstanceId: instanceId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func cancelledEventProvider(worldId world.Id, characterId uint32, routeId uuid.UUID, instanceId uuid.UUID, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &it.Event[it.CancelledEventBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        it.EventTypeCancelled,
		Body: it.CancelledEventBody{
			RouteId:    routeId,
			InstanceId: instanceId,
			Reason:     reason,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
