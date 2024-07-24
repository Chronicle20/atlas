package character

import (
	"atlas-character/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func createdEventProvider(tenant tenant.Model, characterId uint32, worldId byte, name string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[statusEventCreatedBody]{
		Tenant:      tenant,
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        EventCharacterStatusTypeCreated,
		Body: statusEventCreatedBody{
			Name: name,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func loginEventProvider(tenant tenant.Model, characterId uint32, worldId byte, channelId byte, mapId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[statusEventLoginBody]{
		Tenant:      tenant,
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

func logoutEventProvider(tenant tenant.Model, characterId uint32, worldId byte, channelId byte, mapId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[statusEventLogoutBody]{
		Tenant:      tenant,
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

func mapChangedEventProvider(tenant tenant.Model, characterId uint32, worldId byte, channelId byte, oldMapId uint32, targetMapId uint32, targetPortalId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &statusEvent[statusEventMapChangedBody]{
		Tenant:      tenant,
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
