package instance

import (
	character2 "atlas-party-quests/kafka/message/character"
	mapKafka "atlas-party-quests/kafka/message/map"
	pq "atlas-party-quests/kafka/message/party_quest"
	reactorMessage "atlas-party-quests/kafka/message/reactor"
	"atlas-party-quests/kafka/message/system_message"

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

func warpCharacterProvider(worldId world.Id, channelId channel.Id, characterId uint32, mapId _map.Id, instance uuid.UUID) model.Provider[[]kafka.Message] {
	return changeMapProvider(worldId, channelId, characterId, mapId, instance, 0)
}

func awardExperienceProvider(worldId world.Id, channelId channel.Id, characterId uint32, distributions []character2.ExperienceDistributions) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.Command[character2.AwardExperienceCommandBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        character2.CommandAwardExperience,
		Body: character2.AwardExperienceCommandBody{
			ChannelId:     channelId,
			Distributions: distributions,
		},
	}
	return producer.SingleMessageProvider(key, value)
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

func characterLeftEventProvider(worldId world.Id, instanceId uuid.UUID, questId string, characterId uint32, channelId channel.Id, reason string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &pq.StatusEvent[pq.CharacterLeftEventBody]{
		WorldId:    worldId,
		InstanceId: instanceId,
		QuestId:    questId,
		Type:       pq.EventTypeCharacterLeft,
		Body: pq.CharacterLeftEventBody{
			CharacterId: characterId,
			ChannelId:   channelId,
			Reason:      reason,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func sendMessageProvider(worldId world.Id, channelId channel.Id, characterId uint32, messageType string, msg string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &system_message.Command[system_message.SendMessageBody]{
		TransactionId: uuid.Nil,
		WorldId:       worldId,
		ChannelId:     channelId,
		CharacterId:   characterId,
		Type:          system_message.CommandSendMessage,
		Body: system_message.SendMessageBody{
			MessageType: messageType,
			Message:     msg,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func destroyReactorsInFieldProvider(worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := &reactorMessage.Command[reactorMessage.DestroyInFieldCommandBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		Instance:  instance,
		Type:      reactorMessage.CommandTypeDestroyInField,
		Body:      reactorMessage.DestroyInFieldCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func bonusEnteredEventProvider(worldId world.Id, instanceId uuid.UUID, questId string, mapId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(0))
	value := &pq.StatusEvent[pq.BonusEnteredEventBody]{
		WorldId:    worldId,
		InstanceId: instanceId,
		QuestId:    questId,
		Type:       pq.EventTypeBonusEntered,
		Body: pq.BonusEnteredEventBody{
			MapId: mapId,
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

func weatherStartCommandProvider(worldId world.Id, channelId channel.Id, mapId _map.Id, instance uuid.UUID, itemId uint32, message string, durationMs uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := &mapKafka.Command[mapKafka.WeatherStartCommandBody]{
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		Instance:  instance,
		Type:      mapKafka.CommandTypeWeatherStart,
		Body: mapKafka.WeatherStartCommandBody{
			ItemId:     itemId,
			Message:    message,
			DurationMs: durationMs,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
