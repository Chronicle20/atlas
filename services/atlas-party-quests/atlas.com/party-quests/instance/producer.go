package instance

import (
	character2 "atlas-party-quests/kafka/message/character"
	pq "atlas-party-quests/kafka/message/party_quest"

	"github.com/Chronicle20/atlas-constants/channel"
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

func warpCharacterProvider(worldId world.Id, channelId channel.Id, characterId uint32, mapId _map.Id) model.Provider[[]kafka.Message] {
	return changeMapProvider(worldId, channelId, characterId, mapId, uuid.Nil, 0)
}

func instanceCreatedEventProvider(worldId world.Id, instanceId uuid.UUID, questId string, partyId uint32, channelId channel.Id) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(partyId))
	value := &pq.StatusEvent[pq.InstanceCreatedEventBody]{
		WorldId:    worldId,
		InstanceId: instanceId,
		QuestId:    questId,
		Type:       pq.EventTypeInstanceCreated,
		Body: pq.InstanceCreatedEventBody{
			PartyId:   partyId,
			ChannelId: byte(channelId),
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func registrationOpenedEventProvider(worldId world.Id, instanceId uuid.UUID, questId string, duration int64) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(0))
	value := &pq.StatusEvent[pq.RegistrationOpenedEventBody]{
		WorldId:    worldId,
		InstanceId: instanceId,
		QuestId:    questId,
		Type:       pq.EventTypeRegistrationOpened,
		Body: pq.RegistrationOpenedEventBody{
			Duration: duration,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func startedEventProvider(worldId world.Id, instanceId uuid.UUID, questId string, stageIndex uint32, mapIds []uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(0))
	value := &pq.StatusEvent[pq.StartedEventBody]{
		WorldId:    worldId,
		InstanceId: instanceId,
		QuestId:    questId,
		Type:       pq.EventTypeStarted,
		Body: pq.StartedEventBody{
			StageIndex: stageIndex,
			MapIds:     mapIds,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func stageClearedEventProvider(worldId world.Id, instanceId uuid.UUID, questId string, stageIndex uint32, channelId channel.Id, mapIds []uint32, fieldInstances []uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(0))
	value := &pq.StatusEvent[pq.StageClearedEventBody]{
		WorldId:    worldId,
		InstanceId: instanceId,
		QuestId:    questId,
		Type:       pq.EventTypeStageCleared,
		Body: pq.StageClearedEventBody{
			StageIndex:     stageIndex,
			ChannelId:      channelId,
			MapIds:         mapIds,
			FieldInstances: fieldInstances,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func stageAdvancedEventProvider(worldId world.Id, instanceId uuid.UUID, questId string, stageIndex uint32, mapIds []uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(0))
	value := &pq.StatusEvent[pq.StageAdvancedEventBody]{
		WorldId:    worldId,
		InstanceId: instanceId,
		QuestId:    questId,
		Type:       pq.EventTypeStageAdvanced,
		Body: pq.StageAdvancedEventBody{
			StageIndex: stageIndex,
			MapIds:     mapIds,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func completedEventProvider(worldId world.Id, instanceId uuid.UUID, questId string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(0))
	value := &pq.StatusEvent[pq.CompletedEventBody]{
		WorldId:    worldId,
		InstanceId: instanceId,
		QuestId:    questId,
		Type:       pq.EventTypeCompleted,
		Body:       pq.CompletedEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func failedEventProvider(worldId world.Id, instanceId uuid.UUID, questId string, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(0))
	value := &pq.StatusEvent[pq.FailedEventBody]{
		WorldId:    worldId,
		InstanceId: instanceId,
		QuestId:    questId,
		Type:       pq.EventTypeFailed,
		Body: pq.FailedEventBody{
			Reason: reason,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func characterRegisteredEventProvider(worldId world.Id, instanceId uuid.UUID, questId string, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &pq.StatusEvent[pq.CharacterRegisteredEventBody]{
		WorldId:    worldId,
		InstanceId: instanceId,
		QuestId:    questId,
		Type:       pq.EventTypeCharacterRegistered,
		Body: pq.CharacterRegisteredEventBody{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func instanceDestroyedEventProvider(worldId world.Id, instanceId uuid.UUID, questId string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(0))
	value := &pq.StatusEvent[pq.InstanceDestroyedEventBody]{
		WorldId:    worldId,
		InstanceId: instanceId,
		QuestId:    questId,
		Type:       pq.EventTypeInstanceDestroyed,
		Body:       pq.InstanceDestroyedEventBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
