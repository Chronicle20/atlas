package character

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createdEventProvider(characterId uint32, worldId byte, name string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[statusEventCreatedBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        EventCharacterStatusTypeCreated,
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
		Type:        EventCharacterStatusTypeLogin,
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
		Type:        EventCharacterStatusTypeLogout,
		Body: statusEventLogoutBody{
			ChannelId: channelId,
			MapId:     mapId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func mapChangedEventProvider(characterId uint32, worldId byte, channelId byte, oldMapId uint32, targetMapId uint32, targetPortalId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[statusEventMapChangedBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        EventCharacterStatusTypeMapChanged,
		Body: statusEventMapChangedBody{
			ChannelId:      channelId,
			OldMapId:       oldMapId,
			TargetMapId:    targetMapId,
			TargetPortalId: targetPortalId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func move(worldId byte, channelId byte, mapId uint32, characterId uint32, m movement) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &movementCommand{
		WorldId:     worldId,
		ChannelId:   channelId,
		MapId:       mapId,
		CharacterId: characterId,
		Movement:    m,
	}
	return producer.SingleMessageProvider(key, value)
}
