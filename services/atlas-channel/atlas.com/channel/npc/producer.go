package npc

import (
	"github.com/Chronicle20/atlas-kafka/producer"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func startConversationCommandProvider(worldId byte, channelId byte, mapId uint32, npcId uint32, characterId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &command[startConversationCommandBody]{
		WorldId:     worldId,
		ChannelId:   channelId,
		MapId:       mapId,
		NpcId:       npcId,
		CharacterId: characterId,
		Type:        CommandTypeStartConversation,
		Body:        startConversationCommandBody{},
	}
	return producer.SingleMessageProvider(key, value)
}
