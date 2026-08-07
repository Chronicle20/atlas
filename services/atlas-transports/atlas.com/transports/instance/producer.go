package instance

import (
	character2 "atlas-transports/kafka/message/character"
	"atlas-transports/kafka/message/consumable"
	it "atlas-transports/kafka/message/instance_transport"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
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

func transitEnteredEventProvider(worldId world.Id, channelId channel.Id, characterId uint32, routeId uuid.UUID, instanceId uuid.UUID, durationSeconds uint32, message string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &it.Event[it.TransitEnteredEventBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        it.EventTypeTransitEntered,
		Body: it.TransitEnteredEventBody{
			RouteId:         routeId,
			InstanceId:      instanceId,
			ChannelId:       channelId,
			DurationSeconds: durationSeconds,
			Message:         message,
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

// applyConsumableEffectProvider and cancelConsumableEffectProvider build the
// two COMMAND_TOPIC_CONSUMABLE messages this service emits.
//
// TransactionId is deliberately uuid.Nil: atlas-saga-orchestrator treats a nil
// transaction id on the resulting EFFECT_APPLIED event as a non-saga effect
// application and skips saga completion. A fresh uuid would look like an
// orphaned transaction instead.
//
// MapId/Instance are left zero. APPLY ignores the envelope's field entirely
// and resolves the character's live map itself; CANCEL builds a field from the
// envelope but it reaches atlas-buffs' Cancel, which reads only worldId. This
// is what lets the logout path — which has no map in hand — cancel correctly.
func applyConsumableEffectProvider(worldId world.Id, channelId channel.Id, characterId uint32, itemId item.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &consumable.Command[consumable.ApplyConsumableEffectBody]{
		TransactionId: uuid.Nil,
		WorldId:       worldId,
		ChannelId:     channelId,
		CharacterId:   character.Id(characterId),
		Type:          consumable.CommandApplyConsumableEffect,
		Body: consumable.ApplyConsumableEffectBody{
			ItemId: itemId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func cancelConsumableEffectProvider(worldId world.Id, channelId channel.Id, characterId uint32, itemId item.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &consumable.Command[consumable.CancelConsumableEffectBody]{
		TransactionId: uuid.Nil,
		WorldId:       worldId,
		ChannelId:     channelId,
		CharacterId:   character.Id(characterId),
		Type:          consumable.CommandCancelConsumableEffect,
		Body: consumable.CancelConsumableEffectBody{
			ItemId: itemId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
