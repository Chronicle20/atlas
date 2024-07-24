package _map

import (
	"atlas-maps/tenant"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func enterMapProvider(tenant tenant.Model, worldId byte, channelId byte, mapId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := &statusEvent[characterEnter]{
		Tenant:    tenant,
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		Type:      EventTopicMapStatusTypeCharacterEnter,
		Body: characterEnter{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func exitMapProvider(tenant tenant.Model, worldId byte, channelId byte, mapId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(mapId))
	value := &statusEvent[characterExit]{
		Tenant:    tenant,
		WorldId:   worldId,
		ChannelId: channelId,
		MapId:     mapId,
		Type:      EventTopicMapStatusTypeCharacterExit,
		Body: characterExit{
			CharacterId: characterId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
