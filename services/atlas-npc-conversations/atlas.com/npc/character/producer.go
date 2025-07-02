package character

import (
	character2 "atlas-npc-conversations/kafka/message/character"
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func requestChangeMesoCommandProvider(characterId uint32, worldId byte, amount int32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.CommandEvent[character2.RequestChangeMesoBody]{
		CharacterId: characterId,
		WorldId:     worldId,
		Type:        character2.CommandRequestChangeMeso,
		Body: character2.RequestChangeMesoBody{
			Amount: amount,
		},
	}
	return producer.SingleMessageProvider(key, value)
}

func changeMapProvider(worldId byte, channelId byte, characterId uint32, mapId uint32, portalId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &character2.CommandEvent[character2.ChangeMapBody]{
		WorldId:     worldId,
		CharacterId: characterId,
		Type:        character2.CommandChangeMap,
		Body: character2.ChangeMapBody{
			ChannelId: channelId,
			MapId:     mapId,
			PortalId:  portalId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
