package character

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func awardLevelCommandProvider(characterId uint32, worldId byte, channelId byte, amount byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[awardLevelCommandBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        CommandAwardLevel,
		Body: awardLevelCommandBody{
			ChannelId: channelId,
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func createdEventProvider(characterId uint32, worldId byte, name string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[statusEventCreatedBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        StatusEventTypeCreated,
		Body: statusEventCreatedBody{
			Name: name,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func loginEventProvider(characterId uint32, worldId byte, channelId byte, mapId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[statusEventLoginBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        StatusEventTypeLogin,
		Body: statusEventLoginBody{
			ChannelId: channelId,
			MapId:     mapId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func logoutEventProvider(characterId uint32, worldId byte, channelId byte, mapId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[statusEventLogoutBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        StatusEventTypeLogout,
		Body: statusEventLogoutBody{
			ChannelId: channelId,
			MapId:     mapId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeChannelEventProvider(characterId uint32, worldId byte, channelId byte, oldChannelId byte, mapId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[changeChannelEventLoginBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        StatusEventTypeChannelChanged,
		Body: changeChannelEventLoginBody{
			ChannelId:    channelId,
			OldChannelId: oldChannelId,
			MapId:        mapId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func mapChangedEventProvider(characterId uint32, worldId byte, channelId byte, oldMapId uint32, targetMapId uint32, targetPortalId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[statusEventMapChangedBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        StatusEventTypeMapChanged,
		Body: statusEventMapChangedBody{
			ChannelId:      channelId,
			OldMapId:       oldMapId,
			TargetMapId:    targetMapId,
			TargetPortalId: targetPortalId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func jobChangedEventProvider(characterId uint32, worldId byte, channelId byte, jobId uint16) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[jobChangedStatusEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        StatusEventTypeJobChanged,
		Body: jobChangedStatusEventBody{
			ChannelId: channelId,
			JobId:     jobId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func experienceChangedEventProvider(characterId uint32, worldId byte, channelId byte, experience []ExperienceModel, current uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))

	ds := make([]experienceDistributions, 0)
	for _, e := range experience {
		ds = append(ds, experienceDistributions{
			ExperienceType: e.experienceType,
			Amount:         e.amount,
			Attr1:          e.attr1,
		})
	}

	value := &statusEvent[experienceChangedStatusEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        StatusEventTypeExperienceChanged,
		Body: experienceChangedStatusEventBody{
			ChannelId:     channelId,
			Current:       current,
			Distributions: ds,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func levelChangedEventProvider(characterId uint32, worldId byte, channelId byte, amount byte, current byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[levelChangedStatusEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        StatusEventTypeLevelChanged,
		Body: levelChangedStatusEventBody{
			ChannelId: channelId,
			Amount:    amount,
			Current:   current,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func deletedEventProvider(characterId uint32, worldId byte) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[statusEventDeletedBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        StatusEventTypeDeleted,
		Body:        statusEventDeletedBody{},
	}
	return producer.SingleMessageProvider(key, value)
}

func mesoChangedStatusEventProvider(characterId uint32, worldId byte, amount int32, actorId uint32, actorType string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[mesoChangedStatusEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        StatusEventTypeMesoChanged,
		Body: mesoChangedStatusEventBody{
			ActorId:   actorId,
			ActorType: actorType,
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func notEnoughMesoErrorStatusEventProvider(characterId uint32, worldId byte, amount int32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[statusEventErrorBody[notEnoughMesoErrorStatusBodyBody]]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        StatusEventTypeError,
		Body: statusEventErrorBody[notEnoughMesoErrorStatusBodyBody]{
			Error: StatusEventErrorTypeNotEnoughMeso,
			Body: notEnoughMesoErrorStatusBodyBody{
				Amount: amount,
			},
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func fameChangedStatusEventProvider(characterId uint32, worldId byte, amount int8, actorId uint32, actorType string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[fameChangedStatusEventBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        StatusEventTypeFameChanged,
		Body: fameChangedStatusEventBody{
			ActorId:   actorId,
			ActorType: actorType,
			Amount:    amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func statChangedProvider(worldId byte, channelId byte, characterId uint32, updates []string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[statusEventStatChangedBody]{
		CharacterId: characterId,
		Type:        StatusEventTypeStatChanged,
		WorldId:     worldId,
		Body: statusEventStatChangedBody{
			ChannelId:       channelId,
			ExclRequestSent: true,
			Updates:         updates,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
